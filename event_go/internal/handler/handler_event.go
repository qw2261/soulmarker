package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateEventRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.OrganizerID <= 0 {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "门店不能为空")
		return
	}
	org, err := h.store.GetOrganizer(req.OrganizerID)
	if err != nil {
		writeInternalError(w, "create_event_get_organizer", err)
		return
	}
	if org == nil {
		writeError(w, http.StatusBadRequest, api.CodeOrganizerNotFound, model.ErrOrganizerNotFound.Error())
		return
	}

	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "活动标题不能为空")
		return
	}
	if req.EventTime == "" {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "活动时间不能为空")
		return
	}
	if _, err := time.Parse(model.TimeFormat, req.EventTime); err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "活动"+timeParseMsg)
		return
	}
	if strings.TrimSpace(req.Location) == "" {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "活动地点不能为空")
		return
	}
	if req.Capacity <= 0 {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "活动容量必须大于 0")
		return
	}
	if req.Price < 0 {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "价格不能为负数")
		return
	}

	event := &model.Event{
		OrganizerID:   req.OrganizerID,
		OrganizerName: org.Name,
		Title:         req.Title,
		Description:   req.Description,
		EventTime:     req.EventTime,
		Location:      req.Location,
		Capacity:      req.Capacity,
		Price:         req.Price,
	}
	if err := h.store.CreateEvent(event); err != nil {
		writeInternalError(w, "create_event", err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.Response{Code: 201, Message: "活动创建成功", Data: dto.Event(event)})
}

func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	organizerID, _ := strconv.ParseInt(r.URL.Query().Get("organizer_id"), 10, 64)
	page, pageSize := parsePagination(r)

	events, total, err := h.store.ListEvents(model.ListEventsParams{
		Status:      r.URL.Query().Get("status"),
		PriceType:   r.URL.Query().Get("price_type"),
		Keyword:     r.URL.Query().Get("q"),
		OrganizerID: organizerID,
		Offset:      (page - 1) * pageSize,
		Limit:       pageSize,
	})
	if err != nil {
		writeInternalError(w, "list_events", err)
		return
	}
	paginatedOK(w, dto.Events(events), total, page, pageSize)
}

func (h *Handler) GetEvent(w http.ResponseWriter, r *http.Request) {
	id, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}

	event, ok := h.getEventOr404(w, id)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, dto.Response{Code: 200, Message: "ok", Data: dto.Event(event)})
}

func (h *Handler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	id, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}

	var req dto.UpdateEventRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.OrganizerID != nil {
		if *req.OrganizerID <= 0 {
			writeError(w, http.StatusBadRequest, api.CodeValidationError, "门店不能为空")
			return
		}
		organizer, err := h.store.GetOrganizer(*req.OrganizerID)
		if err != nil {
			writeInternalError(w, "update_event_get_organizer", err)
			return
		}
		if organizer == nil {
			writeError(w, http.StatusBadRequest, api.CodeOrganizerNotFound, model.ErrOrganizerNotFound.Error())
			return
		}
	}

	if req.Status != nil {
		valid := map[string]bool{"draft": true, "published": true, "cancelled": true, "ended": true}
		if !valid[*req.Status] {
			writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的状态值，可选: draft, published, cancelled, ended")
			return
		}
	}
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "活动标题不能为空")
		return
	}
	if req.EventTime != nil {
		if _, err := time.Parse(model.TimeFormat, *req.EventTime); err != nil {
			writeError(w, http.StatusBadRequest, api.CodeValidationError, "活动"+timeParseMsg)
			return
		}
	}
	if req.Location != nil && strings.TrimSpace(*req.Location) == "" {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "活动地点不能为空")
		return
	}
	if req.Capacity != nil && *req.Capacity <= 0 {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "活动容量必须大于 0")
		return
	}
	if req.Price != nil && *req.Price < 0 {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "价格不能为负数")
		return
	}

	event, err := h.store.UpdateEvent(id, req.Command())
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, api.CodeEventNotFound, err.Error())
		} else {
			writeInternalError(w, "update_event", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, dto.Response{Code: 200, Message: "活动更新成功", Data: dto.Event(event)})
}

func (h *Handler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	id, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}

	if err := h.store.DeleteEvent(id); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, api.CodeEventNotFound, err.Error())
		} else {
			writeInternalError(w, "delete_event", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, dto.Response{Code: 200, Message: "活动已删除"})
}
