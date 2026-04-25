package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

func parseEventID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func parsePostID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("postId"), 10, 64)
}

func parseTicketID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("ticketId"), 10, 64)
}

func (h *Handler) getEventOr404(w http.ResponseWriter, eventID int64) (*Event, bool) {
	event, err := h.store.GetEvent(eventID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return nil, false
	}
	if event == nil {
		writeJSON(w, http.StatusNotFound, APIResp{Code: 404, Message: ErrNotFound.Error()})
		return nil, false
	}
	return event, true
}

func (h *Handler) checkRegistration(w http.ResponseWriter, eventID int64, contact string) bool {
	registered, err := h.store.IsRegistered(eventID, contact)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return false
	}
	if !registered {
		writeJSON(w, http.StatusForbidden, APIResp{Code: 403, Message: ErrNotRegistered.Error()})
		return false
	}
	return true
}

func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req CreateEventReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}

	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "活动标题不能为空"})
		return
	}
	if req.EventTime == "" {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "活动时间不能为空"})
		return
	}
	if _, err := time.Parse(timeFormat, req.EventTime); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "活动" + timeParseMsg})
		return
	}
	if req.Location == "" {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "活动地点不能为空"})
		return
	}
	if req.Capacity <= 0 {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "活动容量必须大于 0"})
		return
	}
	if req.Price < 0 {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "价格不能为负数"})
		return
	}

	event := &Event{
		Title:       req.Title,
		Description: req.Description,
		EventTime:   req.EventTime,
		Location:    req.Location,
		Capacity:    req.Capacity,
		Price:       req.Price,
	}
	if err := h.store.CreateEvent(event); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, APIResp{Code: 201, Message: "活动创建成功", Data: event})
}

func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	priceType := r.URL.Query().Get("price_type")
	keyword := r.URL.Query().Get("q")

	events, err := h.store.ListEvents(status, priceType, keyword)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIResp{Code: 200, Message: "ok", Data: events})
}

func (h *Handler) GetEvent(w http.ResponseWriter, r *http.Request) {
	id, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	event, ok := h.getEventOr404(w, id)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, APIResp{Code: 200, Message: "ok", Data: event})
}

func (h *Handler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	id, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	var req UpdateEventReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}

	if req.Status != nil {
		valid := map[string]bool{"draft": true, "published": true, "cancelled": true, "ended": true}
		if !valid[*req.Status] {
			writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "无效的状态值，可选: draft, published, cancelled, ended"})
			return
		}
	}

	event, err := h.store.UpdateEvent(id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, APIResp{Code: 404, Message: err.Error()})
		} else {
			writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusOK, APIResp{Code: 200, Message: "活动更新成功", Data: event})
}

func (h *Handler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	id, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	if err := h.store.DeleteEvent(id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, APIResp{Code: 404, Message: err.Error()})
		} else {
			writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusOK, APIResp{Code: 200, Message: "活动已删除"})
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	event, ok := h.getEventOr404(w, eventID)
	if !ok {
		return
	}
	if event.Status != "published" {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "活动未发布，暂无法报名"})
		return
	}

	var req RegisterReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "姓名不能为空"})
		return
	}
	if req.Contact == "" {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "联系方式不能为空"})
		return
	}

	reg := &Registration{
		EventID:  eventID,
		Name:     req.Name,
		Contact:  req.Contact,
		TicketID: req.TicketID,
	}
	if err := h.store.Register(reg); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeJSON(w, http.StatusNotFound, APIResp{Code: 404, Message: err.Error()})
		case errors.Is(err, ErrDuplicate):
			writeJSON(w, http.StatusConflict, APIResp{Code: 409, Message: err.Error()})
		case errors.Is(err, ErrFull):
			writeJSON(w, http.StatusConflict, APIResp{Code: 409, Message: fmt.Sprintf("活动报名已满（上限 %d 人）", event.Capacity)})
		case errors.Is(err, ErrTicketNotFound):
			writeJSON(w, http.StatusNotFound, APIResp{Code: 404, Message: err.Error()})
		case errors.Is(err, ErrTicketSoldOut):
			writeJSON(w, http.StatusConflict, APIResp{Code: 409, Message: err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusCreated, APIResp{Code: 201, Message: "报名成功", Data: reg})
}

func (h *Handler) ListRegistrations(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	_, ok := h.getEventOr404(w, eventID)
	if !ok {
		return
	}

	registrations, err := h.store.ListRegistrations(eventID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, APIResp{Code: 200, Message: "ok", Data: registrations})
}

func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	_, ok := h.getEventOr404(w, eventID)
	if !ok {
		return
	}

	var req CreatePostReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}

	if req.AuthorContact == "" {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "联系方式不能为空"})
		return
	}
	if !h.checkRegistration(w, eventID, req.AuthorContact) {
		return
	}
	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "帖子标题不能为空"})
		return
	}
	if req.Content == "" {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "帖子内容不能为空"})
		return
	}

	post := &Post{
		EventID:       eventID,
		AuthorName:    req.AuthorName,
		AuthorContact: req.AuthorContact,
		Title:         req.Title,
		Content:       req.Content,
	}
	if err := h.store.CreatePost(post); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, APIResp{Code: 201, Message: "发帖成功", Data: post})
}

func (h *Handler) ListPosts(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	_, ok := h.getEventOr404(w, eventID)
	if !ok {
		return
	}

	posts, err := h.store.ListPosts(eventID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, APIResp{Code: 200, Message: "ok", Data: posts})
}

func (h *Handler) GetPost(w http.ResponseWriter, r *http.Request) {
	postID, err := parsePostID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "无效的帖子 ID"})
		return
	}

	post, err := h.store.GetPost(postID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return
	}
	if post == nil {
		writeJSON(w, http.StatusNotFound, APIResp{Code: 404, Message: "帖子不存在"})
		return
	}

	replies, err := h.store.ListReplies(postID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return
	}

	result := map[string]interface{}{
		"post":    post,
		"replies": replies,
	}

	writeJSON(w, http.StatusOK, APIResp{Code: 200, Message: "ok", Data: result})
}

func (h *Handler) CreateReply(w http.ResponseWriter, r *http.Request) {
	postID, err := parsePostID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "无效的帖子 ID"})
		return
	}

	post, err := h.store.GetPost(postID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return
	}
	if post == nil {
		writeJSON(w, http.StatusNotFound, APIResp{Code: 404, Message: "帖子不存在"})
		return
	}

	var req CreateReplyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}

	if req.AuthorContact == "" {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "联系方式不能为空"})
		return
	}
	if !h.checkRegistration(w, post.EventID, req.AuthorContact) {
		return
	}
	if req.Content == "" {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "回复内容不能为空"})
		return
	}

	reply := &Reply{
		PostID:        postID,
		AuthorName:    req.AuthorName,
		AuthorContact: req.AuthorContact,
		Content:       req.Content,
	}
	if err := h.store.CreateReply(reply); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, APIResp{Code: 201, Message: "回复成功", Data: reply})
}

func (h *Handler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	_, ok := h.getEventOr404(w, eventID)
	if !ok {
		return
	}

	var req CreateTicketReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "门票名称不能为空"})
		return
	}
	if req.Price < 0 {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "价格不能为负数"})
		return
	}
	if req.Stock <= 0 {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "库存必须大于 0"})
		return
	}

	ticket := &Ticket{
		EventID: eventID,
		Name:    req.Name,
		Price:   req.Price,
		Stock:   req.Stock,
	}
	if err := h.store.CreateTicket(ticket); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, APIResp{Code: 201, Message: "门票创建成功", Data: ticket})
}

func (h *Handler) ListTickets(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	_, ok := h.getEventOr404(w, eventID)
	if !ok {
		return
	}

	tickets, err := h.store.ListTickets(eventID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, APIResp{Code: 200, Message: "ok", Data: tickets})
}

func (h *Handler) GetTicket(w http.ResponseWriter, r *http.Request) {
	ticketID, err := parseTicketID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "无效的门票 ID"})
		return
	}

	ticket, err := h.store.GetTicket(ticketID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return
	}
	if ticket == nil {
		writeJSON(w, http.StatusNotFound, APIResp{Code: 404, Message: ErrTicketNotFound.Error()})
		return
	}

	writeJSON(w, http.StatusOK, APIResp{Code: 200, Message: "ok", Data: ticket})
}

func (h *Handler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	ticketID, err := parseTicketID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "无效的门票 ID"})
		return
	}

	var req UpdateTicketReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}

	ticket, err := h.store.UpdateTicket(ticketID, req)
	if err != nil {
		if errors.Is(err, ErrTicketNotFound) {
			writeJSON(w, http.StatusNotFound, APIResp{Code: 404, Message: err.Error()})
		} else {
			writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusOK, APIResp{Code: 200, Message: "门票更新成功", Data: ticket})
}

func (h *Handler) DeleteTicket(w http.ResponseWriter, r *http.Request) {
	ticketID, err := parseTicketID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "无效的门票 ID"})
		return
	}

	if err := h.store.DeleteTicket(ticketID); err != nil {
		if errors.Is(err, ErrTicketNotFound) {
			writeJSON(w, http.StatusNotFound, APIResp{Code: 404, Message: err.Error()})
		} else {
			writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusOK, APIResp{Code: 200, Message: "门票已删除"})
}

func cors(next http.Handler) http.Handler {
	allowedOrigin := os.Getenv("CORS_ORIGIN")
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

func writeJSON(w http.ResponseWriter, status int, resp APIResp) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func adminAuth(next http.Handler) http.Handler {
	token := getAdminToken()
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		expected := "Bearer " + token
		if auth != expected {
			writeJSON(w, http.StatusUnauthorized, APIResp{Code: 401, Message: ErrUnauthorized.Error()})
			return
		}
		next.ServeHTTP(w, r)
	})
}
