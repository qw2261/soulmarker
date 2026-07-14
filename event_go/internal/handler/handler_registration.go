package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/config"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}

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
	if !decodeJSON(w, r, &req) {
		return
	}

	userID := user.ID
	reg := &model.Registration{
		EventID:  eventID,
		UserID:   &userID,
		Name:     user.Name,
		Contact:  user.Contact,
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
			writeInternalError(w, "register_event", err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, model.APIResp{Code: 201, Message: "报名成功", Data: reg})
}

func (h *Handler) CancelRegistration(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}

	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	event, ok := h.getEventOr404(w, eventID)
	if !ok {
		return
	}

	if r.ContentLength != 0 {
		var body struct{}
		if !decodeJSON(w, r, &body) {
			return
		}
	}

	eventTime, err := time.Parse(model.TimeFormat, event.EventTime)
	if err != nil {
		writeInternalError(w, "cancel_registration_parse_event_time", err)
		return
	}

	cfg := config.Load()
	deadline := eventTime.Add(-time.Duration(cfg.CancelDeadlineHours) * time.Hour)
	if time.Now().After(deadline) {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: model.ErrCancelDeadlineExceeded.Error()})
		return
	}

	if err := h.store.CancelRegistrationByUserID(eventID, user.ID); err != nil {
		switch {
		case errors.Is(err, model.ErrNotRegistered):
			writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: err.Error()})
		default:
			writeInternalError(w, "cancel_registration", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "已取消报名"})
}

func (h *Handler) GetRegistrationStatus(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	if _, ok := h.getEventOr404(w, eventID); !ok {
		return
	}

	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}

	registered, err := h.store.IsRegisteredByUserID(eventID, user.ID)
	if err != nil {
		writeInternalError(w, "get_registration_status", err)
		return
	}

	writeJSON(w, http.StatusOK, model.APIResp{
		Code:    200,
		Message: "ok",
		Data:    model.RegistrationStatusResp{Registered: registered},
	})
}

func (h *Handler) ListMyRegistrations(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}

	page, pageSize := parsePagination(r)
	registrations, total, err := h.store.ListMyRegistrations(user.ID, (page-1)*pageSize, pageSize)
	if err != nil {
		writeInternalError(w, "list_my_registrations", err)
		return
	}
	paginatedOK(w, registrations, total, page, pageSize)
}

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
		writeInternalError(w, "list_registrations", err)
		return
	}

	paginatedOK(w, registrations, total, page, pageSize)
}
