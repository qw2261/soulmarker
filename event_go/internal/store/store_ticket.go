package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func (s *Store) CreateTicket(t *model.Ticket) error {
	event, err := s.GetEvent(t.EventID)
	if err != nil {
		return err
	}
	if event == nil {
		return model.ErrNotFound
	}
	return s.CreateTicketForOrganization(event.OrganizationID, t)
}

func (s *Store) CreateTicketForOrganization(organizationID int64, t *model.Ticket) error {
	now := time.Now().UTC().Format(model.TimeFormat)
	result, err := s.db.Exec(
		`INSERT INTO tickets (event_id, name, price, stock, created_at, updated_at)
		 SELECT e.id, ?, ?, ?, ?, ? FROM events e
		 WHERE e.id = ? AND e.organization_id = ?`,
		t.Name, t.Price, t.Stock, now, now, t.EventID, organizationID,
	)
	if err != nil {
		return fmt.Errorf("创建门票失败: %w", err)
	}
	created, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("读取门票创建结果失败: %w", err)
	}
	if created != 1 {
		return model.ErrNotFound
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取门票 ID 失败: %w", err)
	}
	t.ID = id
	t.CreatedAt, _ = time.Parse(model.TimeFormat, now)
	t.UpdatedAt = t.CreatedAt
	return nil
}

func (s *Store) ListTickets(eventID int64, offset, limit int) ([]*model.Ticket, int, error) {
	event, err := s.GetEvent(eventID)
	if err != nil {
		return nil, 0, err
	}
	if event == nil {
		return []*model.Ticket{}, 0, nil
	}
	return s.ListTicketsForOrganization(event.OrganizationID, eventID, offset, limit)
}

func (s *Store) ListTicketsForOrganization(organizationID, eventID int64, offset, limit int) ([]*model.Ticket, int, error) {
	var total int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM tickets t JOIN events e ON e.id = t.event_id
		 WHERE t.event_id = ? AND e.organization_id = ?`, eventID, organizationID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询门票总数失败: %w", err)
	}

	query := `SELECT t.id, t.event_id, t.name, t.price, t.stock, t.created_at, t.updated_at
			 FROM tickets t JOIN events e ON e.id = t.event_id
			 WHERE t.event_id = ? AND e.organization_id = ? ORDER BY t.created_at ASC`
	args := []interface{}{eventID, organizationID}
	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询门票列表失败: %w", err)
	}
	defer rows.Close()

	var tickets []*model.Ticket
	for rows.Next() {
		t := &model.Ticket{}
		var createdAt, updatedAt string
		if err := rows.Scan(&t.ID, &t.EventID, &t.Name, &t.Price, &t.Stock, &createdAt, &updatedAt); err != nil {
			return nil, 0, fmt.Errorf("读取门票记录失败: %w", err)
		}
		createdAtTime, err := time.Parse(model.TimeFormat, createdAt)
		if err != nil {
			return nil, 0, fmt.Errorf("解析门票创建时间失败: %w", err)
		}
		t.CreatedAt = createdAtTime
		updatedAtTime, err := time.Parse(model.TimeFormat, updatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("解析门票更新时间失败: %w", err)
		}
		t.UpdatedAt = updatedAtTime
		tickets = append(tickets, t)
	}

	if tickets == nil {
		tickets = []*model.Ticket{}
	}
	return tickets, total, nil
}

func (s *Store) GetTicket(ticketID int64) (*model.Ticket, error) {
	return s.getTicket(0, 0, ticketID)
}

func (s *Store) GetTicketForOrganization(organizationID, eventID, ticketID int64) (*model.Ticket, error) {
	return s.getTicket(organizationID, eventID, ticketID)
}

func (s *Store) getTicket(organizationID, eventID, ticketID int64) (*model.Ticket, error) {
	t := &model.Ticket{}
	var createdAt, updatedAt string
	query := `SELECT t.id, t.event_id, t.name, t.price, t.stock, t.created_at, t.updated_at
		 FROM tickets t`
	args := []interface{}{ticketID}
	if organizationID > 0 {
		query += ` JOIN events e ON e.id = t.event_id
			 WHERE t.id = ? AND t.event_id = ? AND e.organization_id = ?`
		args = []interface{}{ticketID, eventID, organizationID}
	} else {
		query += ` WHERE t.id = ?`
	}
	err := s.db.QueryRow(query, args...).Scan(&t.ID, &t.EventID, &t.Name, &t.Price, &t.Stock, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询门票失败: %w", err)
	}
	createdAtTime, err := time.Parse(model.TimeFormat, createdAt)
	if err != nil {
		return nil, fmt.Errorf("解析门票创建时间失败: %w", err)
	}
	t.CreatedAt = createdAtTime
	updatedAtTime, err := time.Parse(model.TimeFormat, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("解析门票更新时间失败: %w", err)
	}
	t.UpdatedAt = updatedAtTime
	return t, nil
}

func (s *Store) UpdateTicket(id int64, req model.UpdateTicketReq) (*model.Ticket, error) {
	ticket, err := s.GetTicket(id)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, model.ErrTicketNotFound
	}
	event, err := s.GetEvent(ticket.EventID)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, model.ErrTicketNotFound
	}
	return s.UpdateTicketForOrganization(event.OrganizationID, ticket.EventID, id, req)
}

func (s *Store) UpdateTicketForOrganization(organizationID, eventID, id int64, req model.UpdateTicketReq) (*model.Ticket, error) {
	ticket, err := s.GetTicketForOrganization(organizationID, eventID, id)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, model.ErrTicketNotFound
	}

	if req.Name != nil {
		ticket.Name = *req.Name
	}
	if req.Price != nil {
		ticket.Price = *req.Price
	}
	if req.Stock != nil {
		ticket.Stock = *req.Stock
	}

	now := time.Now().UTC().Format(model.TimeFormat)
	result, err := s.db.Exec(
		`UPDATE tickets SET name=?, price=?, stock=?, updated_at=?
		 WHERE id=? AND event_id IN (SELECT id FROM events WHERE id = ? AND organization_id = ?)`,
		ticket.Name, ticket.Price, ticket.Stock, now, id, eventID, organizationID,
	)
	if err != nil {
		return nil, fmt.Errorf("更新门票失败: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("读取门票更新结果失败: %w", err)
	}
	if updated != 1 {
		return nil, model.ErrTicketNotFound
	}

	ticket.UpdatedAt, _ = time.Parse(model.TimeFormat, now)
	return ticket, nil
}

func (s *Store) DeleteTicket(id int64) error {
	ticket, err := s.GetTicket(id)
	if err != nil {
		return err
	}
	if ticket == nil {
		return model.ErrTicketNotFound
	}
	event, err := s.GetEvent(ticket.EventID)
	if err != nil {
		return err
	}
	if event == nil {
		return model.ErrTicketNotFound
	}
	return s.DeleteTicketForOrganization(event.OrganizationID, ticket.EventID, id)
}

func (s *Store) DeleteTicketForOrganization(organizationID, eventID, id int64) error {
	s.registrationMu.Lock()
	defer s.registrationMu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开启删除门票事务失败: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`UPDATE registrations SET ticket_id = NULL
		 WHERE ticket_id IN (
			 SELECT t.id FROM tickets t JOIN events e ON e.id = t.event_id
			 WHERE t.id = ? AND t.event_id = ? AND e.organization_id = ?
		 )`, id, eventID, organizationID,
	); err != nil {
		return fmt.Errorf("解除报名门票引用失败: %w", err)
	}
	result, err := tx.Exec(
		`DELETE FROM tickets WHERE id = ? AND event_id IN (
			 SELECT id FROM events WHERE id = ? AND organization_id = ?
		 )`, id, eventID, organizationID,
	)
	if err != nil {
		return fmt.Errorf("删除门票失败: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取删除影响行数失败: %w", err)
	}
	if n == 0 {
		return model.ErrTicketNotFound
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交删除门票事务失败: %w", err)
	}
	return nil
}
