package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func (h *Handler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}

	organizationID, ok := h.organizationIDForManagedEvent(w, r, eventID)
	if !ok {
		return
	}

	var req dto.CreateTicketRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "门票名称不能为空")
		return
	}
	if req.Price < 0 {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "价格不能为负数")
		return
	}
	if req.Stock <= 0 {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "库存必须大于 0")
		return
	}

	ticket := &model.Ticket{
		EventID: eventID,
		Name:    req.Name,
		Price:   req.Price,
		Stock:   req.Stock,
	}
	if err := h.operations.CreateTicket(organizationID, ticket); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, api.CodeEventNotFound, err.Error())
		} else {
			writeInternalError(w, "create_ticket", err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, dto.Response{Code: 201, Message: "门票创建成功", Data: dto.Ticket(ticket)})
}

func (h *Handler) ListOrganizationTickets(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}
	organizationID, ok := h.organizationIDForManagedEvent(w, r, eventID)
	if !ok {
		return
	}
	page, pageSize := parsePagination(r)
	tickets, total, err := h.operations.ListTickets(organizationID, eventID, (page-1)*pageSize, pageSize)
	if err != nil {
		writeInternalError(w, "list_organization_tickets", err)
		return
	}
	if event, err := h.operations.GetEvent(organizationID, eventID); err != nil {
		writeInternalError(w, "get_organization_ticket_event", err)
		return
	} else if event == nil {
		writeError(w, http.StatusNotFound, api.CodeEventNotFound, model.ErrNotFound.Error())
		return
	}
	paginatedOK(w, dto.Tickets(tickets), total, page, pageSize)
}

func (h *Handler) GetOrganizationTicket(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}
	ticketID, err := parseTicketID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的门票 ID")
		return
	}
	organizationID, ok := h.organizationIDForManagedEvent(w, r, eventID)
	if !ok {
		return
	}
	ticket, err := h.operations.GetTicket(organizationID, eventID, ticketID)
	if err != nil {
		writeInternalError(w, "get_organization_ticket", err)
		return
	}
	if ticket == nil {
		writeError(w, http.StatusNotFound, api.CodeTicketNotFound, model.ErrTicketNotFound.Error())
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: 200, Message: "ok", Data: dto.Ticket(ticket)})
}

func (h *Handler) ListTickets(w http.ResponseWriter, r *http.Request) {
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

	tickets, total, err := h.store.ListTickets(eventID, offset, pageSize)
	if err != nil {
		writeInternalError(w, "list_tickets", err)
		return
	}

	paginatedOK(w, dto.Tickets(tickets), total, page, pageSize)
}

func (h *Handler) GetTicket(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}

	ticketID, err := parseTicketID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的门票 ID")
		return
	}

	ticket, ok := h.getTicketForEventOr404(w, eventID, ticketID)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, dto.Response{Code: 200, Message: "ok", Data: dto.Ticket(ticket)})
}

func (h *Handler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}
	organizationID, ok := h.organizationIDForManagedEvent(w, r, eventID)
	if !ok {
		return
	}

	ticketID, err := parseTicketID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的门票 ID")
		return
	}

	var req dto.UpdateTicketRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "门票名称不能为空")
		return
	}
	if req.Price != nil && *req.Price < 0 {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "价格不能为负数")
		return
	}
	if req.Stock != nil && *req.Stock < 0 {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "库存不能为负数")
		return
	}

	ticket, err := h.operations.UpdateTicket(organizationID, eventID, ticketID, req.Command())
	if err != nil {
		if errors.Is(err, model.ErrTicketNotFound) {
			writeError(w, http.StatusNotFound, api.CodeTicketNotFound, err.Error())
		} else {
			writeInternalError(w, "update_ticket", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, dto.Response{Code: 200, Message: "门票更新成功", Data: dto.Ticket(ticket)})
}

func (h *Handler) DeleteTicket(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}
	organizationID, ok := h.organizationIDForManagedEvent(w, r, eventID)
	if !ok {
		return
	}

	ticketID, err := parseTicketID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的门票 ID")
		return
	}

	if err := h.operations.DeleteTicket(organizationID, eventID, ticketID); err != nil {
		if errors.Is(err, model.ErrTicketNotFound) {
			writeError(w, http.StatusNotFound, api.CodeTicketNotFound, err.Error())
		} else {
			writeInternalError(w, "delete_ticket", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, dto.Response{Code: 200, Message: "门票已删除"})
}
