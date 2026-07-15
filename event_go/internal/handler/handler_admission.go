package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func (h *Handler) GetMyAdmission(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}
	eventID, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}
	admission, err := h.admissions.GetForUser(eventID, user.ID)
	if err != nil {
		if errors.Is(err, model.ErrAdmissionNotFound) {
			writeError(w, http.StatusNotFound, api.CodeAdmissionNotFound, err.Error())
		} else {
			writeInternalError(w, "get_my_admission", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: 200, Message: "ok", Data: dto.MyAdmission(admission)})
}

func (h *Handler) ListMyAdmissions(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}
	page, pageSize := parsePagination(r)
	admissions, total, err := h.admissions.ListForUser(user.ID, (page-1)*pageSize, pageSize)
	if err != nil {
		writeInternalError(w, "list_my_admissions", err)
		return
	}
	paginatedOK(w, dto.MyAdmissions(admissions), total, page, pageSize)
}

func (h *Handler) CheckIn(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}
	if _, ok := h.getEventOr404(w, eventID); !ok {
		return
	}
	var request dto.CheckinRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.Credential) == "" {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "入场凭证不能为空")
		return
	}
	checkin, duplicate, err := h.admissions.CheckIn(eventID, request.Credential, "platform_admin")
	if err != nil {
		switch {
		case errors.Is(err, model.ErrAdmissionNotFound):
			writeError(w, http.StatusNotFound, api.CodeAdmissionNotFound, err.Error())
		case errors.Is(err, model.ErrAdmissionRevoked):
			writeError(w, http.StatusConflict, api.CodeAdmissionRevoked, err.Error())
		default:
			writeInternalError(w, "check_in", err)
		}
		return
	}
	status := http.StatusCreated
	message := "核销成功"
	if duplicate {
		status = http.StatusOK
		message = "该凭证已核销"
	}
	writeJSON(w, status, dto.Response{
		Code: status, Message: message,
		Data: dto.CheckinResultResponse{Checkin: dto.Checkin(checkin), AlreadyCheckedIn: duplicate},
	})
}

func (h *Handler) ListCheckins(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}
	if _, ok := h.getEventOr404(w, eventID); !ok {
		return
	}
	page, pageSize := parsePagination(r)
	checkins, total, err := h.admissions.ListCheckins(eventID, (page-1)*pageSize, pageSize)
	if err != nil {
		writeInternalError(w, "list_checkins", err)
		return
	}
	paginatedOK(w, dto.Checkins(checkins), total, page, pageSize)
}
