package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func (s *Store) Register(r *model.Registration) error {
	event, err := s.GetEvent(r.EventID)
	if err != nil {
		return err
	}
	if event == nil {
		return model.ErrNotFound
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM registrations WHERE event_id = ?`, r.EventID).Scan(&count); err != nil {
		return fmt.Errorf("查询报名人数失败: %w", err)
	}
	if count >= event.Capacity {
		return model.ErrFull
	}

	ticketName := ""
	if r.TicketID != nil {
		var stock int
		var name string
		err := tx.QueryRow(
			`SELECT name, stock FROM tickets WHERE id = ? AND event_id = ?`,
			*r.TicketID, r.EventID,
		).Scan(&name, &stock)
		if err == sql.ErrNoRows {
			return model.ErrTicketNotFound
		}
		if err != nil {
			return fmt.Errorf("查询门票失败: %w", err)
		}
		if stock <= 0 {
			return model.ErrTicketSoldOut
		}
		result, err := tx.Exec(
			`UPDATE tickets SET stock = stock - 1, updated_at = ? WHERE id = ? AND stock > 0`,
			time.Now().UTC().Format(model.TimeFormat), *r.TicketID,
		)
		if err != nil {
			return fmt.Errorf("扣减门票库存失败: %w", err)
		}
		n, _ := result.RowsAffected()
		if n == 0 {
			return model.ErrTicketSoldOut
		}
		ticketName = name
	}

	now := time.Now().UTC().Format(model.TimeFormat)
	result, err := tx.Exec(
		`INSERT INTO registrations (event_id, name, contact, ticket_id, ticket_name, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		r.EventID, r.Name, r.Contact, r.TicketID, ticketName, now,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return model.ErrDuplicate
		}
		return fmt.Errorf("报名失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取报名 ID 失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	r.ID = id
	r.TicketName = ticketName
	createdAt, _ := time.Parse(model.TimeFormat, now)
	r.CreatedAt = createdAt
	return nil
}

func (s *Store) ListRegistrations(eventID int64, offset, limit int) ([]*model.Registration, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM registrations WHERE event_id = ?`, eventID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询报名总数失败: %w", err)
	}

	query := `SELECT id, event_id, name, contact, ticket_id, ticket_name, created_at
		 FROM registrations WHERE event_id = ? ORDER BY created_at ASC`
	args := []interface{}{eventID}
	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询报名列表失败: %w", err)
	}
	defer rows.Close()

	var registrations []*model.Registration
	for rows.Next() {
		r := &model.Registration{}
		var createdAt string
		if err := rows.Scan(&r.ID, &r.EventID, &r.Name, &r.Contact, &r.TicketID, &r.TicketName, &createdAt); err != nil {
			return nil, 0, fmt.Errorf("读取报名记录失败: %w", err)
		}
		createdAtTime, err := time.Parse(model.TimeFormat, createdAt)
		if err != nil {
			return nil, 0, fmt.Errorf("解析报名时间失败: %w", err)
		}
		r.CreatedAt = createdAtTime
		registrations = append(registrations, r)
	}

	if registrations == nil {
		registrations = []*model.Registration{}
	}
	return registrations, total, nil
}

func (s *Store) IsRegistered(eventID int64, contact string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM registrations WHERE event_id = ? AND contact = ?`,
		eventID, contact,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("查询报名信息失败: %w", err)
	}
	return count > 0, nil
}

func (s *Store) CancelRegistration(eventID int64, contact string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	var ticketID sql.NullInt64
	err = tx.QueryRow(
		`SELECT ticket_id FROM registrations WHERE event_id = ? AND contact = ?`,
		eventID, contact,
	).Scan(&ticketID)
	if err == sql.ErrNoRows {
		return model.ErrNotRegistered
	}
	if err != nil {
		return fmt.Errorf("查询报名记录失败: %w", err)
	}

	if ticketID.Valid {
		_, err = tx.Exec(
			`UPDATE tickets SET stock = stock + 1, updated_at = ? WHERE id = ?`,
			time.Now().UTC().Format(model.TimeFormat), ticketID.Int64,
		)
		if err != nil {
			return fmt.Errorf("退还门票库存失败: %w", err)
		}
	}

	_, err = tx.Exec(
		`DELETE FROM registrations WHERE event_id = ? AND contact = ?`,
		eventID, contact,
	)
	if err != nil {
		return fmt.Errorf("删除报名记录失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}
