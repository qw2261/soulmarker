package handler

import (
	"encoding/json"
	"errors"
	"fmt"
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

// CreateEvent 处理创建活动请求，需要管理员权限
func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req model.CreateEventReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}

	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "活动标题不能为空"})
		return
	}
	if req.EventTime == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "活动时间不能为空"})
		return
	}
	if _, err := time.Parse(model.TimeFormat, req.EventTime); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "活动" + timeParseMsg})
		return
	}
	if req.Location == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "活动地点不能为空"})
		return
	}
	if req.Capacity <= 0 {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "活动容量必须大于 0"})
		return
	}
	if req.Price < 0 {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "价格不能为负数"})
		return
	}

	event := &model.Event{
		Title:       req.Title,
		Description: req.Description,
		EventTime:   req.EventTime,
		Location:    req.Location,
		Capacity:    req.Capacity,
		Price:       req.Price,
	}
	if err := h.store.CreateEvent(event); err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, model.APIResp{Code: 201, Message: "活动创建成功", Data: event})
}

// ListEvents 返回活动列表，支持按status、price_type、q(关键字)筛选，支持分页
func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	priceType := r.URL.Query().Get("price_type")
	keyword := r.URL.Query().Get("q")
	page, pageSize := parsePagination(r)
	offset := (page - 1) * pageSize

	events, total, err := h.store.ListEvents(status, priceType, keyword, offset, pageSize)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}
	paginatedOK(w, events, total, page, pageSize)
}

// GetEvent 获取单个活动详情
func (h *Handler) GetEvent(w http.ResponseWriter, r *http.Request) {
	id, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	event, ok := h.getEventOr404(w, id)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "ok", Data: event})
}

// UpdateEvent 更新活动信息，支持更新status/title/description等字段，需要管理员权限
func (h *Handler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	id, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	var req model.UpdateEventReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}

	if req.Status != nil {
		valid := map[string]bool{"draft": true, "published": true, "cancelled": true, "ended": true}
		if !valid[*req.Status] {
			writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的状态值，可选: draft, published, cancelled, ended"})
			return
		}
	}

	event, err := h.store.UpdateEvent(id, req)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: err.Error()})
		} else {
			writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "活动更新成功", Data: event})
}

// DeleteEvent 删除活动及其关联的报名、帖子、门票等数据，需要管理员权限
func (h *Handler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	id, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	if err := h.store.DeleteEvent(id); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: err.Error()})
		} else {
			writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "活动已删除"})
}

// Register 处理用户报名活动请求，支持选择门票
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	event, ok := h.getEventOr404(w, eventID)
	if !ok {
		return
	}
	if event.Status != "published" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "活动未发布，暂无法报名"})
		return
	}

	var req model.RegisterReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "姓名不能为空"})
		return
	}
	if req.Contact == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "联系方式不能为空"})
		return
	}

	reg := &model.Registration{
		EventID:  eventID,
		Name:     req.Name,
		Contact:  req.Contact,
		TicketID: req.TicketID,
	}
	if err := h.store.Register(reg); err != nil {
		switch {
		case errors.Is(err, model.ErrNotFound):
			writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: err.Error()})
		case errors.Is(err, model.ErrDuplicate):
			writeJSON(w, http.StatusConflict, model.APIResp{Code: 409, Message: err.Error()})
		case errors.Is(err, model.ErrFull):
			writeJSON(w, http.StatusConflict, model.APIResp{Code: 409, Message: fmt.Sprintf("活动报名已满（上限 %d 人）", event.Capacity)})
		case errors.Is(err, model.ErrTicketNotFound):
			writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: err.Error()})
		case errors.Is(err, model.ErrTicketSoldOut):
			writeJSON(w, http.StatusConflict, model.APIResp{Code: 409, Message: err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusCreated, model.APIResp{Code: 201, Message: "报名成功", Data: reg})
}

// CancelRegistration 取消报名，活动开始前N小时可自由取消（N通过CANCEL_DEADLINE_HOURS配置，默认24）
func (h *Handler) CancelRegistration(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	event, ok := h.getEventOr404(w, eventID)
	if !ok {
		return
	}

	var req model.CancelRegistrationReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}

	if req.Contact == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "联系方式不能为空"})
		return
	}

	eventTime, err := time.Parse(model.TimeFormat, event.EventTime)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: "解析活动时间失败"})
		return
	}

	cfg := config.Load()
	deadline := eventTime.Add(-time.Duration(cfg.CancelDeadlineHours) * time.Hour)
	if time.Now().After(deadline) {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: model.ErrCancelDeadlineExceeded.Error()})
		return
	}

	if err := h.store.CancelRegistration(eventID, req.Contact); err != nil {
		switch {
		case errors.Is(err, model.ErrNotRegistered):
			writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "已取消报名"})
}

// ListRegistrations 返回活动的报名记录，支持分页
func (h *Handler) ListRegistrations(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	_, ok := h.getEventOr404(w, eventID)
	if !ok {
		return
	}

	page, pageSize := parsePagination(r)
	offset := (page - 1) * pageSize

	registrations, total, err := h.store.ListRegistrations(eventID, offset, pageSize)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}

	paginatedOK(w, registrations, total, page, pageSize)
}

// CreatePost 创建活动帖子，需要用户已报名
func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	_, ok := h.getEventOr404(w, eventID)
	if !ok {
		return
	}

	var req model.CreatePostReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}

	if req.AuthorContact == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "联系方式不能为空"})
		return
	}
	if !h.checkRegistration(w, eventID, req.AuthorContact) {
		return
	}
	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "帖子标题不能为空"})
		return
	}
	if req.Content == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "帖子内容不能为空"})
		return
	}

	post := &model.Post{
		EventID:       eventID,
		AuthorName:    req.AuthorName,
		AuthorContact: req.AuthorContact,
		Title:         req.Title,
		Content:       req.Content,
	}
	if err := h.store.CreatePost(post); err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, model.APIResp{Code: 201, Message: "发帖成功", Data: post})
}

func (h *Handler) ListPosts(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	_, ok := h.getEventOr404(w, eventID)
	if !ok {
		return
	}

	page, pageSize := parsePagination(r)
	offset := (page - 1) * pageSize

	posts, total, err := h.store.ListPosts(eventID, offset, pageSize)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}

	paginatedOK(w, posts, total, page, pageSize)
}

func (h *Handler) GetPost(w http.ResponseWriter, r *http.Request) {
	postID, err := parsePostID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的帖子 ID"})
		return
	}

	post, err := h.store.GetPost(postID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}
	if post == nil {
		writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: "帖子不存在"})
		return
	}

	replies, err := h.store.ListReplies(postID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}

	result := map[string]interface{}{
		"post":    post,
		"replies": replies,
	}

	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "ok", Data: result})
}

// CreateReply 创建帖子回复，需要用户已报名对应活动
func (h *Handler) CreateReply(w http.ResponseWriter, r *http.Request) {
	postID, err := parsePostID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的帖子 ID"})
		return
	}

	post, err := h.store.GetPost(postID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}
	if post == nil {
		writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: "帖子不存在"})
		return
	}

	var req model.CreateReplyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}

	if req.AuthorContact == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "联系方式不能为空"})
		return
	}
	if !h.checkRegistration(w, post.EventID, req.AuthorContact) {
		return
	}
	if req.Content == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "回复内容不能为空"})
		return
	}

	reply := &model.Reply{
		PostID:        postID,
		AuthorName:    req.AuthorName,
		AuthorContact: req.AuthorContact,
		Content:       req.Content,
	}
	if err := h.store.CreateReply(reply); err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, model.APIResp{Code: 201, Message: "回复成功", Data: reply})
}

// CreateTicket 创建活动门票类型，需要管理员权限
func (h *Handler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	_, ok := h.getEventOr404(w, eventID)
	if !ok {
		return
	}

	var req model.CreateTicketReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "门票名称不能为空"})
		return
	}
	if req.Price < 0 {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "价格不能为负数"})
		return
	}
	if req.Stock <= 0 {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "库存必须大于 0"})
		return
	}

	ticket := &model.Ticket{
		EventID: eventID,
		Name:    req.Name,
		Price:   req.Price,
		Stock:   req.Stock,
	}
	if err := h.store.CreateTicket(ticket); err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, model.APIResp{Code: 201, Message: "门票创建成功", Data: ticket})
}

// ListTickets 返回活动的门票类型，支持分页
func (h *Handler) ListTickets(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	_, ok := h.getEventOr404(w, eventID)
	if !ok {
		return
	}

	page, pageSize := parsePagination(r)
	offset := (page - 1) * pageSize

	tickets, total, err := h.store.ListTickets(eventID, offset, pageSize)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}

	paginatedOK(w, tickets, total, page, pageSize)
}

// GetTicket 获取单个门票详情
func (h *Handler) GetTicket(w http.ResponseWriter, r *http.Request) {
	ticketID, err := parseTicketID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的门票 ID"})
		return
	}

	ticket, err := h.store.GetTicket(ticketID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}
	if ticket == nil {
		writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: model.ErrTicketNotFound.Error()})
		return
	}

	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "ok", Data: ticket})
}

// UpdateTicket 更新门票信息，需要管理员权限
func (h *Handler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	ticketID, err := parseTicketID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的门票 ID"})
		return
	}

	var req model.UpdateTicketReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}

	ticket, err := h.store.UpdateTicket(ticketID, req)
	if err != nil {
		if errors.Is(err, model.ErrTicketNotFound) {
			writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: err.Error()})
		} else {
			writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "门票更新成功", Data: ticket})
}

// DeleteTicket 删除门票，需要管理员权限
func (h *Handler) DeleteTicket(w http.ResponseWriter, r *http.Request) {
	ticketID, err := parseTicketID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的门票 ID"})
		return
	}

	if err := h.store.DeleteTicket(ticketID); err != nil {
		if errors.Is(err, model.ErrTicketNotFound) {
			writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: err.Error()})
		} else {
			writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "门票已删除"})
}

// HealthHandler 健康检查接口，返回服务状态和数据库连接状态
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
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

// AdminAuth 中间件，验证管理员Bearer令牌，保护需要管理员权限的API
func AdminAuth(next http.Handler) http.Handler {
	token := getAdminToken()
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		expected := "Bearer " + token
		if auth != expected {
			writeJSON(w, http.StatusUnauthorized, model.APIResp{Code: 401, Message: model.ErrUnauthorized.Error()})
			return
		}
		next.ServeHTTP(w, r)
	})
}
