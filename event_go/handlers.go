package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
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
	events, err := h.store.ListEvents()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIResp{Code: 200, Message: "ok", Data: events})
}

func (h *Handler) GetEvent(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	event, err := h.store.GetEvent(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return
	}
	if event == nil {
		writeJSON(w, http.StatusNotFound, APIResp{Code: 404, Message: ErrNotFound.Error()})
		return
	}

	writeJSON(w, http.StatusOK, APIResp{Code: 200, Message: "ok", Data: event})
}

func (h *Handler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
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
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
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
	idStr := r.PathValue("id")
	eventID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	event, err := h.store.GetEvent(eventID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return
	}
	if event == nil {
		writeJSON(w, http.StatusNotFound, APIResp{Code: 404, Message: ErrNotFound.Error()})
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
		EventID: eventID,
		Name:    req.Name,
		Contact: req.Contact,
	}
	if err := h.store.Register(reg); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeJSON(w, http.StatusNotFound, APIResp{Code: 404, Message: err.Error()})
		case errors.Is(err, ErrDuplicate):
			writeJSON(w, http.StatusConflict, APIResp{Code: 409, Message: err.Error()})
		case errors.Is(err, ErrFull):
			writeJSON(w, http.StatusConflict, APIResp{Code: 409, Message: fmt.Sprintf("活动报名已满（上限 %d 人）", event.Capacity)})
		default:
			writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusCreated, APIResp{Code: 201, Message: "报名成功", Data: reg})
}

func (h *Handler) ListRegistrations(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	eventID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	event, err := h.store.GetEvent(eventID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return
	}
	if event == nil {
		writeJSON(w, http.StatusNotFound, APIResp{Code: 404, Message: ErrNotFound.Error()})
		return
	}

	registrations, err := h.store.ListRegistrations(eventID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResp{Code: 500, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, APIResp{Code: 200, Message: "ok", Data: registrations})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
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
