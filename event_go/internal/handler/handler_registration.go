package handler

import (
	"errors"
	"net/http"

	"github.com/qw2261/soulmarker/event_go/internal/model"
	"github.com/qw2261/soulmarker/event_go/internal/service"
)

func (h *Handler) getRegistrationEventOr404(w http.ResponseWriter, eventID int64) (*model.Event, bool) {
	event, err := h.registrations.GetEvent(eventID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: model.ErrNotFound.Error()})
		} else {
			writeInternalError(w, "get_registration_event", err)
		}
		return nil, false
	}
	return event, true
}

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

	event, ok := h.getRegistrationEventOr404(w, eventID)
	if !ok {
		return
	}

	var req model.RegisterReq
	if !decodeJSON(w, r, &req) {
		return
	}

	registration, err := h.registrations.Register(event, user, req.TicketID)
	if err != nil {
		var fullError *service.RegistrationFullError
		switch {
		case errors.Is(err, model.ErrNotFound):
			writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: err.Error()})
		case errors.Is(err, service.ErrEventNotPublished):
			writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: err.Error()})
		case errors.Is(err, model.ErrDuplicate):
			writeJSON(w, http.StatusConflict, model.APIResp{Code: 409, Message: err.Error()})
		case errors.As(err, &fullError):
			writeJSON(w, http.StatusConflict, model.APIResp{Code: 409, Message: fullError.Error()})
		case errors.Is(err, model.ErrTicketNotFound):
			writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: err.Error()})
		case errors.Is(err, model.ErrTicketSoldOut):
			writeJSON(w, http.StatusConflict, model.APIResp{Code: 409, Message: err.Error()})
		default:
			writeInternalError(w, "register_event", err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, model.APIResp{Code: 201, Message: "报名成功", Data: registration})
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

	event, ok := h.getRegistrationEventOr404(w, eventID)
	if !ok {
		return
	}

	if r.ContentLength != 0 {
		var body struct{}
		if !decodeJSON(w, r, &body) {
			return
		}
	}

	if err := h.registrations.Cancel(event, user.ID); err != nil {
		switch {
		case errors.Is(err, model.ErrCancelDeadlineExceeded):
			writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: err.Error()})
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
