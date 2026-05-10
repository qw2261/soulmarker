package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func (h *Handler) CreateOrganizer(w http.ResponseWriter, r *http.Request) {
	var req model.CreateOrganizerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "门店名称不能为空"})
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
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, model.APIResp{Code: 201, Message: "门店创建成功", Data: o})
}

func (h *Handler) GetOrganizer(w http.ResponseWriter, r *http.Request) {
	id, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的门店 ID"})
		return
	}
	o, err := h.store.GetOrganizer(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}
	if o == nil {
		writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: model.ErrOrganizerNotFound.Error()})
		return
	}
	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "ok", Data: o})
}

func (h *Handler) ListOrganizers(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePagination(r)
	offset := (page - 1) * pageSize
	organizers, total, err := h.store.ListOrganizers(offset, pageSize)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}
	paginatedOK(w, organizers, total, page, pageSize)
}

func (h *Handler) UpdateOrganizer(w http.ResponseWriter, r *http.Request) {
	id, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的门店 ID"})
		return
	}
	var req model.UpdateOrganizerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}
	o, err := h.store.UpdateOrganizer(id, req)
	if err != nil {
		if errors.Is(err, model.ErrOrganizerNotFound) {
			writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: err.Error()})
		} else {
			writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		}
		return
	}
	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "门店更新成功", Data: o})
}

func (h *Handler) DeleteOrganizer(w http.ResponseWriter, r *http.Request) {
	id, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的门店 ID"})
		return
	}
	if err := h.store.DeleteOrganizer(id); err != nil {
		if errors.Is(err, model.ErrOrganizerNotFound) {
			writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: err.Error()})
		} else {
			writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		}
		return
	}
	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "门店已删除"})
}
