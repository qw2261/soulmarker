package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/auth"
	"github.com/qw2261/soulmarker/event_go/internal/authorization"
	"github.com/qw2261/soulmarker/event_go/internal/clock"
	"github.com/qw2261/soulmarker/event_go/internal/config"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/model"
	"github.com/qw2261/soulmarker/event_go/internal/store"
)

const timeParseMsg = "格式错误，请使用 RFC3339 格式，例如：2026-12-31T18:00:00+08:00"

const maxRequestBodyBytes int64 = 1 << 20

// Handler 负责处理HTTP请求，协调store层进行数据操作
type Handler struct {
	store          *store.Store
	config         *config.Config
	clock          clock.Clock
	tokens         auth.TokenManager
	registrations  RegistrationService
	discussions    DiscussionService
	admissions     AdmissionService
	authentication AuthenticationService
	moderation     ContentModerationService
	notifications  NotificationService
	organizations  OrganizationAuthorizationService
	operations     OrganizationOperationsService
	startTime      time.Time
	version        string
}

type RegistrationService interface {
	GetEvent(eventID int64) (*model.Event, error)
	Register(event *model.Event, user *model.User, ticketID *int64) (*model.Registration, error)
	Cancel(event *model.Event, userID int64) error
	IsRegistered(eventID, userID int64) (bool, error)
}

type DiscussionService interface {
	GetEvent(eventID int64) (*model.Event, error)
	GetPost(eventID, postID int64) (*model.Post, error)
	CreatePost(eventID int64, user *model.User, title, content string) (*model.Post, error)
	CreateReply(post *model.Post, user *model.User, content string) (*model.Reply, error)
}

type AdmissionService interface {
	GetForUser(eventID, userID int64) (*model.MyAdmission, error)
	ListForUser(userID int64, offset, limit int) ([]*model.MyAdmission, int, error)
	ListActivitiesForUser(userID int64, offset, limit int) ([]*model.MyActivity, int, error)
	CheckIn(eventID int64, credential, actor string) (*model.Checkin, bool, error)
	ListCheckins(eventID int64, offset, limit int) ([]*model.Checkin, int, error)
}

type AuthenticationService interface {
	RequestPasswordReset(ctx context.Context, contact string) error
	ResetPassword(token, password string) error
	Logout(userID int64) error
	RequestRecoveryEmail(ctx context.Context, user *model.User, email, password string) error
	ConfirmRecoveryEmail(token string) error
}

type ContentModerationService interface {
	ReportPost(eventID, postID int64, user *model.User, category, detail string) (*model.ContentReport, bool, error)
	ReportReply(eventID, postID, replyID int64, user *model.User, category, detail string) (*model.ContentReport, bool, error)
	ListReports(params model.ListContentReportsParams) ([]*model.ContentReport, int, error)
	ResolveReport(id int64, resolution, note, actor string) (*model.ContentReport, error)
	RemovePost(eventID, postID int64, actor, reason string) error
	RestorePost(eventID, postID int64, actor, reason string) error
	RemoveReply(eventID, postID, replyID int64, actor, reason string) error
	RestoreReply(eventID, postID, replyID int64, actor, reason string) error
	ListActions(offset, limit int) ([]*model.ContentModerationAction, int, error)
}

type NotificationService interface {
	List(userID int64, unreadOnly bool, offset, limit int) ([]*model.Notification, int, error)
	UnreadCount(userID int64) (int, error)
	MarkRead(userID, notificationID int64) error
	MarkAllRead(userID int64) (int, error)
}

type OrganizationAuthorizationService interface {
	ListForUser(userID int64) ([]authorization.OrganizationContext, error)
	Authorize(userID, organizationID int64, capability authorization.Capability) (*authorization.OrganizationContext, error)
}

type OrganizationOperationsService interface {
	CreateEvent(organizationID int64, event *model.Event) error
	ListEvents(organizationID int64, params model.ListEventsParams) ([]*model.Event, int, error)
	GetEvent(organizationID, eventID int64) (*model.Event, error)
	UpdateEvent(organizationID, eventID int64, req model.UpdateEventReq) (*model.Event, error)
	DeleteEvent(organizationID, eventID int64) error
	CreateTicket(organizationID int64, ticket *model.Ticket) error
	ListTickets(organizationID, eventID int64, offset, limit int) ([]*model.Ticket, int, error)
	GetTicket(organizationID, eventID, ticketID int64) (*model.Ticket, error)
	UpdateTicket(organizationID, eventID, ticketID int64, req model.UpdateTicketReq) (*model.Ticket, error)
	DeleteTicket(organizationID, eventID, ticketID int64) error
	ListRegistrations(organizationID, eventID int64, offset, limit int) ([]*model.Registration, int, error)
	CheckIn(organizationID, eventID int64, credential, actor string) (*model.Checkin, bool, error)
	ListCheckins(organizationID, eventID int64, offset, limit int) ([]*model.Checkin, int, error)
}

type Dependencies struct {
	Clock          clock.Clock
	Tokens         auth.TokenManager
	Registrations  RegistrationService
	Discussions    DiscussionService
	Admissions     AdmissionService
	Authentication AuthenticationService
	Moderation     ContentModerationService
	Notifications  NotificationService
	Organizations  OrganizationAuthorizationService
	Operations     OrganizationOperationsService
}

// NewHandler 创建 Handler，并显式注入启动配置与难以测试的运行时依赖。
func NewHandler(s *store.Store, cfg *config.Config, dependencies Dependencies) *Handler {
	return &Handler{
		store:          s,
		config:         cfg,
		clock:          dependencies.Clock,
		tokens:         dependencies.Tokens,
		registrations:  dependencies.Registrations,
		discussions:    dependencies.Discussions,
		admissions:     dependencies.Admissions,
		authentication: dependencies.Authentication,
		moderation:     dependencies.Moderation,
		notifications:  dependencies.Notifications,
		organizations:  dependencies.Organizations,
		operations:     dependencies.Operations,
		startTime:      dependencies.Clock.Now(),
		version:        cfg.Version,
	}
}

// parseEventID 从URL路径中解析活动ID
func parseEventID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

// parsePostID 从URL路径中解析帖子ID
func parsePostID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("postId"), 10, 64)
}

func parseReplyID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("replyId"), 10, 64)
}

func parseReportID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("reportId"), 10, 64)
}

func parseNotificationID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("notificationId"), 10, 64)
}

func parseOrganizationID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("organizationId"), 10, 64)
}

// parseTicketID 从URL路径中解析门票ID
func parseTicketID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("ticketId"), 10, 64)
}

// parsePagination 从URL查询参数中解析分页参数，默认 page=1, page_size=20，最大 page_size=100
func parsePagination(r *http.Request) (page, pageSize int) {
	page = 1
	pageSize = 20

	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
			if pageSize > 100 {
				pageSize = 100
			}
		}
	}
	return
}

// paginatedOK 写入带分页信息的成功响应
func paginatedOK(w http.ResponseWriter, data interface{}, total, page, pageSize int) {
	writeJSON(w, http.StatusOK, dto.Response{
		Code:     200,
		Message:  "ok",
		Data:     data,
		Total:    &total,
		Page:     &page,
		PageSize: &pageSize,
	})
}

// getEventOr404 根据ID获取活动，若不存在则写入404响应
func (h *Handler) getEventOr404(w http.ResponseWriter, eventID int64) (*model.Event, bool) {
	event, err := h.store.GetEvent(eventID)
	if err != nil {
		writeInternalError(w, "get_event", err)
		return nil, false
	}
	if event == nil {
		writeError(w, http.StatusNotFound, api.CodeEventNotFound, model.ErrNotFound.Error())
		return nil, false
	}
	return event, true
}

func (h *Handler) organizationIDForManagedEvent(w http.ResponseWriter, r *http.Request, eventID int64) (int64, bool) {
	if value, ok := authorization.OrganizationContextFromContext(r.Context()); ok {
		return value.OrganizationID, true
	}
	event, err := h.store.GetEvent(eventID)
	if err != nil {
		writeInternalError(w, "resolve_event_organization", err)
		return 0, false
	}
	if event == nil {
		writeError(w, http.StatusNotFound, api.CodeEventNotFound, model.ErrNotFound.Error())
		return 0, false
	}
	return event.OrganizationID, true
}

func managedActor(r *http.Request) string {
	if value, ok := authorization.OrganizationContextFromContext(r.Context()); ok {
		return fmt.Sprintf("organization_member:%d", value.UserID)
	}
	return authorization.PrincipalTypePlatformAdmin
}

// getTicketForEventOr404 确保门票存在且属于 URL 指定的活动。
func (h *Handler) getTicketForEventOr404(w http.ResponseWriter, eventID, ticketID int64) (*model.Ticket, bool) {
	ticket, err := h.store.GetTicket(ticketID)
	if err != nil {
		writeInternalError(w, "get_ticket", err)
		return nil, false
	}
	if ticket == nil || ticket.EventID != eventID {
		writeError(w, http.StatusNotFound, api.CodeTicketNotFound, model.ErrTicketNotFound.Error())
		return nil, false
	}
	return ticket, true
}

func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	dbStatus := "connected"
	if err := h.store.Ping(); err != nil {
		dbStatus = "disconnected"
		slog.Error("health check database failure", "error", err)
	}

	status := "ok"
	if dbStatus == "disconnected" {
		status = "degraded"
	}

	data := dto.HealthResponse{
		Status:        status,
		Version:       h.version,
		UptimeSeconds: int64(h.clock.Now().Sub(h.startTime).Seconds()),
		DB:            dbStatus,
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: 200, Message: "ok", Data: data})
}

// CORS 中间件使用启动时注入的允许来源处理跨域请求。
func CORS(next http.Handler, allowedOrigin string) http.Handler {
	if allowedOrigin == "" {
		allowedOrigin = "*"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Admin-Token")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// writeJSON 统一JSON响应格式，设置Content-Type和响应状态码
func writeJSON(w http.ResponseWriter, status int, resp dto.Response) {
	if status >= http.StatusBadRequest && resp.ErrorCode == "" {
		slog.Error("error response missing business error code", "status", status)
		resp = api.NewErrorResponse(status, api.CodeInternalError, "")
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("encode response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, code api.ErrorCode, message string) {
	writeJSON(w, status, api.NewErrorResponse(status, code, message))
}

func writeInternalError(w http.ResponseWriter, operation string, err error) {
	slog.Error("request failed", "operation", operation, "error", err)
	writeError(w, http.StatusInternalServerError, api.CodeInternalError, "")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, api.CodeRequestTooLarge, "")
			return false
		}
		writeError(w, http.StatusBadRequest, api.CodeInvalidJSON, "")
		return false
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, api.CodeInvalidJSON, "请求体只能包含一个 JSON 对象")
		return false
	}
	return true
}

// PlatformAdminAuth 验证平台管理员令牌并写入独立 principal，不授予任何租户角色。
func PlatformAdminAuth(next http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token != "" && r.Header.Get("X-Admin-Token") != token {
			writeError(w, http.StatusUnauthorized, api.CodeAdminAuthInvalid, model.ErrUnauthorized.Error())
			return
		}
		ctx := authorization.WithPlatformAdmin(r.Context())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AdminAuth 保留一个源码兼容周期；新路由必须使用 PlatformAdminAuth 明确语义。
func AdminAuth(next http.Handler, token string) http.Handler {
	return PlatformAdminAuth(next, token)
}
