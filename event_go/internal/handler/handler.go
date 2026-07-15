package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/auth"
	"github.com/qw2261/soulmarker/event_go/internal/clock"
	"github.com/qw2261/soulmarker/event_go/internal/config"
	"github.com/qw2261/soulmarker/event_go/internal/model"
	"github.com/qw2261/soulmarker/event_go/internal/store"
)

const timeParseMsg = "格式错误，请使用 RFC3339 格式，例如：2026-12-31T18:00:00+08:00"

const maxRequestBodyBytes int64 = 1 << 20

// Handler 负责处理HTTP请求，协调store层进行数据操作
type Handler struct {
	store         *store.Store
	config        *config.Config
	clock         clock.Clock
	tokens        auth.TokenManager
	registrations RegistrationService
	discussions   DiscussionService
	startTime     time.Time
	version       string
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

type Dependencies struct {
	Clock         clock.Clock
	Tokens        auth.TokenManager
	Registrations RegistrationService
	Discussions   DiscussionService
}

// NewHandler 创建 Handler，并显式注入启动配置与难以测试的运行时依赖。
func NewHandler(s *store.Store, cfg *config.Config, dependencies Dependencies) *Handler {
	return &Handler{
		store:         s,
		config:        cfg,
		clock:         dependencies.Clock,
		tokens:        dependencies.Tokens,
		registrations: dependencies.Registrations,
		discussions:   dependencies.Discussions,
		startTime:     dependencies.Clock.Now(),
		version:       cfg.Version,
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
	writeJSON(w, http.StatusOK, model.APIResp{
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
		writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: model.ErrNotFound.Error()})
		return nil, false
	}
	return event, true
}

// getTicketForEventOr404 确保门票存在且属于 URL 指定的活动。
func (h *Handler) getTicketForEventOr404(w http.ResponseWriter, eventID, ticketID int64) (*model.Ticket, bool) {
	ticket, err := h.store.GetTicket(ticketID)
	if err != nil {
		writeInternalError(w, "get_ticket", err)
		return nil, false
	}
	if ticket == nil || ticket.EventID != eventID {
		writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: model.ErrTicketNotFound.Error()})
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

	data := map[string]interface{}{
		"status":         status,
		"version":        h.version,
		"uptime_seconds": int64(h.clock.Now().Sub(h.startTime).Seconds()),
		"db":             dbStatus,
	}
	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "ok", Data: data})
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
func writeJSON(w http.ResponseWriter, status int, resp model.APIResp) {
	if status >= http.StatusBadRequest && resp.ErrorCode == "" {
		resp.ErrorCode = defaultErrorCode(status)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("encode response", "error", err)
	}
}

func defaultErrorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusRequestEntityTooLarge:
		return "REQUEST_TOO_LARGE"
	default:
		return "INTERNAL_ERROR"
	}
}

func writeInternalError(w http.ResponseWriter, operation string, err error) {
	slog.Error("request failed", "operation", operation, "error", err)
	writeJSON(w, http.StatusInternalServerError, model.APIResp{
		Code:      http.StatusInternalServerError,
		ErrorCode: "INTERNAL_ERROR",
		Message:   "服务器内部错误",
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, model.APIResp{
				Code:      http.StatusRequestEntityTooLarge,
				ErrorCode: "REQUEST_TOO_LARGE",
				Message:   "请求体过大",
			})
			return false
		}
		writeJSON(w, http.StatusBadRequest, model.APIResp{
			Code:      http.StatusBadRequest,
			ErrorCode: "INVALID_JSON",
			Message:   "请求体格式错误",
		})
		return false
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, model.APIResp{
			Code:      http.StatusBadRequest,
			ErrorCode: "INVALID_JSON",
			Message:   "请求体只能包含一个 JSON 对象",
		})
		return false
	}
	return true
}

// AdminAuth 中间件，验证管理员令牌，保护需要管理员权限的API
func AdminAuth(next http.Handler, token string) http.Handler {
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("X-Admin-Token")
		if auth != token {
			writeJSON(w, http.StatusUnauthorized, model.APIResp{Code: 401, Message: model.ErrUnauthorized.Error()})
			return
		}
		next.ServeHTTP(w, r)
	})
}
