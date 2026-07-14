package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func (h *Handler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	_, ok := h.getEventOr404(w, eventID)
	if !ok {
		return
	}

	var req model.CreateTicketReq
	if !decodeJSON(w, r, &req) {
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "门票名称不能为空"})
		return
	}
	if req.Price < 0 {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "价格不能为负数"})
		return
	}
	if req.Stock <= 0 {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "库存必须大于 0"})
		return
	}

	ticket := &model.Ticket{
		EventID: eventID,
		Name:    req.Name,
		Price:   req.Price,
		Stock:   req.Stock,
	}
	if err := h.store.CreateTicket(ticket); err != nil {
		writeInternalError(w, "create_ticket", err)
		return
	}

	writeJSON(w, http.StatusCreated, model.APIResp{Code: 201, Message: "门票创建成功", Data: ticket})
}

func (h *Handler) ListTickets(w http.ResponseWriter, r *http.Request) {
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

	tickets, total, err := h.store.ListTickets(eventID, offset, pageSize)
	if err != nil {
		writeInternalError(w, "list_tickets", err)
		return
	}

	paginatedOK(w, tickets, total, page, pageSize)
}

func (h *Handler) GetTicket(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	ticketID, err := parseTicketID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的门票 ID"})
		return
	}

	ticket, ok := h.getTicketForEventOr404(w, eventID, ticketID)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "ok", Data: ticket})
}

func (h *Handler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	ticketID, err := parseTicketID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的门票 ID"})
		return
	}

	if _, ok := h.getTicketForEventOr404(w, eventID, ticketID); !ok {
		return
	}

	var req model.UpdateTicketReq
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "门票名称不能为空"})
		return
	}
	if req.Price != nil && *req.Price < 0 {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "价格不能为负数"})
		return
	}
	if req.Stock != nil && *req.Stock < 0 {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "库存不能为负数"})
		return
	}

	ticket, err := h.store.UpdateTicket(ticketID, req)
	if err != nil {
		if errors.Is(err, model.ErrTicketNotFound) {
			writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: err.Error()})
		} else {
			writeInternalError(w, "update_ticket", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "门票更新成功", Data: ticket})
}

func (h *Handler) DeleteTicket(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	ticketID, err := parseTicketID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的门票 ID"})
		return
	}

	if _, ok := h.getTicketForEventOr404(w, eventID, ticketID); !ok {
		return
	}

	if err := h.store.DeleteTicket(ticketID); err != nil {
		if errors.Is(err, model.ErrTicketNotFound) {
			writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: err.Error()})
		} else {
			writeInternalError(w, "delete_ticket", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "门票已删除"})
}
