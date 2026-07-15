package handler

import (
	"errors"
	"net/http"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/model"
	"github.com/qw2261/soulmarker/event_go/internal/service"
)

func (h *Handler) getRegistrationEventOr404(w http.ResponseWriter, eventID int64) (*model.Event, bool) {
	event, err := h.registrations.GetEvent(eventID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, api.CodeEventNotFound, model.ErrNotFound.Error())
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
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}

	event, ok := h.getRegistrationEventOr404(w, eventID)
	if !ok {
		return
	}

	var req dto.RegisterEventRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	registration, err := h.registrations.Register(event, user, req.TicketID)
	if err != nil {
		var fullError *service.RegistrationFullError
		switch {
		case errors.Is(err, model.ErrNotFound):
			writeError(w, http.StatusNotFound, api.CodeEventNotFound, err.Error())
		case errors.Is(err, service.ErrEventNotPublished):
			writeError(w, http.StatusBadRequest, api.CodeEventNotPublished, err.Error())
		case errors.Is(err, model.ErrDuplicate):
			writeError(w, http.StatusConflict, api.CodeRegistrationDuplicate, err.Error())
		case errors.As(err, &fullError):
			writeError(w, http.StatusConflict, api.CodeEventCapacityFull, fullError.Error())
		case errors.Is(err, model.ErrTicketNotFound):
			writeError(w, http.StatusNotFound, api.CodeTicketNotFound, err.Error())
		case errors.Is(err, model.ErrTicketSoldOut):
			writeError(w, http.StatusConflict, api.CodeTicketSoldOut, err.Error())
		default:
			writeInternalError(w, "register_event", err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, dto.Response{Code: 201, Message: "报名成功", Data: dto.Registration(registration)})
}

func (h *Handler) CancelRegistration(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}

	eventID, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
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
			writeError(w, http.StatusBadRequest, api.CodeCancellationDeadlineExceeded, err.Error())
		case errors.Is(err, model.ErrNotRegistered):
			writeError(w, http.StatusNotFound, api.CodeRegistrationNotFound, err.Error())
		case errors.Is(err, model.ErrAdmissionCheckedIn):
			writeError(w, http.StatusConflict, api.CodeAdmissionAlreadyCheckedIn, err.Error())
		default:
			writeInternalError(w, "cancel_registration", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, dto.Response{Code: 200, Message: "已取消报名"})
}

func (h *Handler) GetRegistrationStatus(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}

	if _, ok := h.getEventOr404(w, eventID); !ok {
		return
	}

	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}

	registered, err := h.registrations.IsRegistered(eventID, user.ID)
	if err != nil {
		writeInternalError(w, "get_registration_status", err)
		return
	}
	status := dto.RegistrationStatusResponse{Registered: registered}
	if registered {
		admission, err := h.admissions.GetForUser(eventID, user.ID)
		if err == nil {
			mapped := dto.Admission(&admission.Admission)
			status.Admission = &mapped
		} else if !errors.Is(err, model.ErrAdmissionNotFound) {
			writeInternalError(w, "get_registration_admission", err)
			return
		}
	}

	writeJSON(w, http.StatusOK, dto.Response{
		Code:    200,
		Message: "ok",
		Data:    status,
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
	paginatedOK(w, dto.MyRegistrations(registrations), total, page, pageSize)
}

func (h *Handler) ListRegistrations(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
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

	paginatedOK(w, dto.Registrations(registrations), total, page, pageSize)
}
