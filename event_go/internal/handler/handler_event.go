package handler

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/authorization"
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
	organizationID := org.OrganizationID
	if value, ok := authorization.OrganizationContextFromContext(r.Context()); ok {
		organizationID = value.OrganizationID
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
	coverURL, ok := normalizeEventCoverURL(req.CoverURL)
	if !ok {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "活动封面必须是有效的 HTTP 或 HTTPS 地址")
		return
	}

	event := &model.Event{
		OrganizationID: organizationID,
		OrganizerID:    req.OrganizerID,
		OrganizerName:  org.Name,
		Title:          req.Title,
		Description:    req.Description,
		CoverURL:       coverURL,
		EventTime:      req.EventTime,
		Location:       req.Location,
		Capacity:       req.Capacity,
		Price:          req.Price,
	}
	if err := h.operations.CreateEvent(organizationID, event); err != nil {
		if errors.Is(err, model.ErrOrganizerNotFound) {
			writeError(w, http.StatusBadRequest, api.CodeOrganizerNotFound, model.ErrOrganizerNotFound.Error())
		} else {
			writeInternalError(w, "create_event", err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, dto.Response{Code: 201, Message: "活动创建成功", Data: dto.Event(event)})
}

func (h *Handler) ListOrganizationEvents(w http.ResponseWriter, r *http.Request) {
	value, ok := authorization.OrganizationContextFromContext(r.Context())
	if !ok {
		writeInternalError(w, "organization_context_missing", errors.New("organization context missing"))
		return
	}
	organizerID, _ := strconv.ParseInt(r.URL.Query().Get("organizer_id"), 10, 64)
	page, pageSize := parsePagination(r)
	events, total, err := h.operations.ListEvents(value.OrganizationID, model.ListEventsParams{
		Status:      r.URL.Query().Get("status"),
		PriceType:   r.URL.Query().Get("price_type"),
		Keyword:     r.URL.Query().Get("q"),
		OrganizerID: organizerID,
		Offset:      (page - 1) * pageSize,
		Limit:       pageSize,
	})
	if err != nil {
		writeInternalError(w, "list_organization_events", err)
		return
	}
	paginatedOK(w, dto.Events(events), total, page, pageSize)
}

func (h *Handler) GetOrganizationEvent(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}
	organizationID, ok := h.organizationIDForManagedEvent(w, r, eventID)
	if !ok {
		return
	}
	event, err := h.operations.GetEvent(organizationID, eventID)
	if err != nil {
		writeInternalError(w, "get_organization_event", err)
		return
	}
	if event == nil {
		writeError(w, http.StatusNotFound, api.CodeEventNotFound, model.ErrNotFound.Error())
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: 200, Message: "ok", Data: dto.Event(event)})
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
	organizationID, ok := h.organizationIDForManagedEvent(w, r, id)
	if !ok {
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
	if req.CoverURL != nil {
		coverURL, ok := normalizeEventCoverURL(*req.CoverURL)
		if !ok {
			writeError(w, http.StatusBadRequest, api.CodeValidationError, "活动封面必须是有效的 HTTP 或 HTTPS 地址")
			return
		}
		req.CoverURL = &coverURL
	}
	if req.Price != nil && *req.Price < 0 {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "价格不能为负数")
		return
	}

	event, err := h.operations.UpdateEvent(organizationID, id, req.Command())
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, api.CodeEventNotFound, err.Error())
		} else if errors.Is(err, model.ErrOrganizerNotFound) {
			writeError(w, http.StatusBadRequest, api.CodeOrganizerNotFound, err.Error())
		} else {
			writeInternalError(w, "update_event", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, dto.Response{Code: 200, Message: "活动更新成功", Data: dto.Event(event)})
}

func normalizeEventCoverURL(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", true
	}
	if len(value) > 2048 {
		return "", false
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", false
	}
	return value, true
}

func (h *Handler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	id, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}
	organizationID, ok := h.organizationIDForManagedEvent(w, r, id)
	if !ok {
		return
	}

	if err := h.operations.DeleteEvent(organizationID, id); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, api.CodeEventNotFound, err.Error())
		} else if errors.Is(err, model.ErrEventHasAdmissions) {
			writeError(w, http.StatusConflict, api.CodeEventHasAdmissions, err.Error())
		} else {
			writeInternalError(w, "delete_event", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, dto.Response{Code: 200, Message: "活动已删除"})
}
