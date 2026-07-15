package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func (h *Handler) CreateOrganizer(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateOrganizerRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "门店名称不能为空")
		return
	}

	o := &model.Organizer{
		Name:        req.Name,
		Description: req.Description,
		Contact:     req.Contact,
		LogoURL:     req.LogoURL,
		Address:     req.Address,
		Website:     req.Website,
		Tags:        req.Tags,
	}
	if err := h.store.CreateOrganizer(o); err != nil {
		writeInternalError(w, "create_organizer", err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.Response{Code: 201, Message: "门店创建成功", Data: dto.Organizer(o)})
}

func (h *Handler) GetOrganizer(w http.ResponseWriter, r *http.Request) {
	id, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的门店 ID")
		return
	}
	o, err := h.store.GetOrganizer(id)
	if err != nil {
		writeInternalError(w, "get_organizer", err)
		return
	}
	if o == nil {
		writeError(w, http.StatusNotFound, api.CodeOrganizerNotFound, model.ErrOrganizerNotFound.Error())
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: 200, Message: "ok", Data: dto.Organizer(o)})
}

func (h *Handler) ListOrganizers(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePagination(r)
	offset := (page - 1) * pageSize
	organizers, total, err := h.store.ListOrganizers(offset, pageSize)
	if err != nil {
		writeInternalError(w, "list_organizers", err)
		return
	}
	paginatedOK(w, dto.Organizers(organizers), total, page, pageSize)
}

func (h *Handler) UpdateOrganizer(w http.ResponseWriter, r *http.Request) {
	id, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的门店 ID")
		return
	}
	var req dto.UpdateOrganizerRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "门店名称不能为空")
		return
	}
	o, err := h.store.UpdateOrganizer(id, req.Command())
	if err != nil {
		if errors.Is(err, model.ErrOrganizerNotFound) {
			writeError(w, http.StatusNotFound, api.CodeOrganizerNotFound, err.Error())
		} else {
			writeInternalError(w, "update_organizer", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: 200, Message: "门店更新成功", Data: dto.Organizer(o)})
}

func (h *Handler) DeleteOrganizer(w http.ResponseWriter, r *http.Request) {
	id, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的门店 ID")
		return
	}
	if err := h.store.DeleteOrganizer(id); err != nil {
		if errors.Is(err, model.ErrOrganizerNotFound) {
			writeError(w, http.StatusNotFound, api.CodeOrganizerNotFound, err.Error())
		} else {
			writeInternalError(w, "delete_organizer", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: 200, Message: "门店已删除"})
}
