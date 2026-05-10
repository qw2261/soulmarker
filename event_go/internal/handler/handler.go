package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/config"
	"github.com/qw2261/soulmarker/event_go/internal/model"
	"github.com/qw2261/soulmarker/event_go/internal/store"
)

const timeParseMsg = "格式错误，请使用 RFC3339 格式，例如：2026-12-31T18:00:00+08:00"

// getAdminToken 从环境变量获取管理员令牌，用于保护需要管理员权限的API
func getAdminToken() string {
	return config.Load().AdminToken
}

// Handler 负责处理HTTP请求，协调store层进行数据操作
type Handler struct {
	store     *store.Store
	startTime time.Time
	version   string
}

// NewHandler 创建Handler实例，接收store层指针用于数据访问
func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s, startTime: time.Now(), version: getVersion()}
}

// getVersion 获取服务版本，默认"dev"，可通过VERSION环境变量配置
func getVersion() string {
	return config.Load().Version
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
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return nil, false
	}
	if event == nil {
		writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: model.ErrNotFound.Error()})
		return nil, false
	}
	return event, true
}

// checkRegistration 验证用户是否已报名活动，未报名返回403响应
func (h *Handler) checkRegistration(w http.ResponseWriter, eventID int64, contact string) bool {
	registered, err := h.store.IsRegistered(eventID, contact)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return false
	}
	if !registered {
		writeJSON(w, http.StatusForbidden, model.APIResp{Code: 403, Message: model.ErrNotRegistered.Error()})
		return false
	}
	return true
}

func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	dbStatus := "connected"
	var dbError string
	if err := h.store.Ping(); err != nil {
		dbStatus = "disconnected"
		dbError = err.Error()
	}

	status := "ok"
	if dbStatus == "disconnected" {
		status = "degraded"
	}

	data := map[string]interface{}{
		"status":         status,
		"version":        h.version,
		"uptime_seconds": int64(time.Since(h.startTime).Seconds()),
		"db":             dbStatus,
	}
	if dbError != "" {
		data["db_error"] = dbError
	}

	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "ok", Data: data})
}

// CORS 中间件，处理跨域请求，支持通过CORS_ORIGIN环境变量配置允许的来源
func CORS(next http.Handler) http.Handler {
	cfg := config.Load()
	allowedOrigin := cfg.CORSOrigin
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

// AdminAuth 中间件，验证管理员令牌，保护需要管理员权限的API
func AdminAuth(next http.Handler) http.Handler {
	token := getAdminToken()
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
