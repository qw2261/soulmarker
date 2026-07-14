package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func (s *Store) CreateEvent(e *model.Event) error {
	now := time.Now().UTC().Format(model.TimeFormat)
	result, err := s.db.Exec(
		`INSERT INTO events (organizer_id, title, description, event_time, location, capacity, price, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 'published', ?, ?)`,
		e.OrganizerID, e.Title, e.Description, e.EventTime, e.Location, e.Capacity, e.Price, now, now,
	)
	if err != nil {
		return fmt.Errorf("创建活动失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取活动 ID 失败: %w", err)
	}
	e.ID = id
	e.Status = "published"
	e.CreatedAt, _ = time.Parse(model.TimeFormat, now)
	e.UpdatedAt = e.CreatedAt
	return nil
}

func buildEventsQuery(params model.ListEventsParams) (string, []interface{}) {
	where := " WHERE 1=1"
	var args []interface{}

	if params.OrganizerID > 0 {
		where += " AND e.organizer_id = ?"
		args = append(args, params.OrganizerID)
	}
	if params.Status != "" {
		where += " AND e.status = ?"
		args = append(args, params.Status)
	}
	if params.PriceType == "free" {
		where += " AND e.price = 0"
	} else if params.PriceType == "paid" {
		where += " AND e.price > 0"
	}
	if params.Keyword != "" {
		where += " AND (e.title LIKE ? OR e.description LIKE ?)"
		args = append(args, "%"+params.Keyword+"%", "%"+params.Keyword+"%")
	}
	return where, args
}

func (s *Store) ListEvents(params model.ListEventsParams) ([]*model.Event, int, error) {
	where, args := buildEventsQuery(params)

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := s.db.QueryRow("SELECT COUNT(*) FROM events e"+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询活动总数失败: %w", err)
	}

	query := `SELECT e.id, e.organizer_id, COALESCE(o.name, ''), e.title, e.description, e.event_time, e.location, e.capacity, e.price, e.status, e.created_at, e.updated_at
		FROM events e LEFT JOIN organizers o ON e.organizer_id = o.id` + where + ` ORDER BY e.created_at DESC`
	if params.Limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, params.Limit, params.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询活动列表失败: %w", err)
	}
	defer rows.Close()

	var events []*model.Event
	for rows.Next() {
		e := &model.Event{}
		var createdAt, updatedAt string
		if err := rows.Scan(&e.ID, &e.OrganizerID, &e.OrganizerName, &e.Title, &e.Description, &e.EventTime, &e.Location,
			&e.Capacity, &e.Price, &e.Status, &createdAt, &updatedAt); err != nil {
			return nil, 0, fmt.Errorf("读取活动记录失败: %w", err)
		}
		createdAtTime, err := time.Parse(model.TimeFormat, createdAt)
		if err != nil {
			return nil, 0, fmt.Errorf("解析活动创建时间失败: %w", err)
		}
		e.CreatedAt = createdAtTime
		updatedAtTime, err := time.Parse(model.TimeFormat, updatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("解析活动更新时间失败: %w", err)
		}
		e.UpdatedAt = updatedAtTime
		events = append(events, e)
	}

	if events == nil {
		events = []*model.Event{}
	}
	return events, total, nil
}

func (s *Store) GetEvent(id int64) (*model.Event, error) {
	e := &model.Event{}
	var createdAt, updatedAt string
	err := s.db.QueryRow(
		`SELECT e.id, e.organizer_id, COALESCE(o.name, ''), e.title, e.description, e.event_time, e.location, e.capacity, e.price, e.status, e.created_at, e.updated_at
		 FROM events e LEFT JOIN organizers o ON e.organizer_id = o.id WHERE e.id = ?`, id,
	).Scan(&e.ID, &e.OrganizerID, &e.OrganizerName, &e.Title, &e.Description, &e.EventTime, &e.Location,
		&e.Capacity, &e.Price, &e.Status, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询活动失败: %w", err)
	}
	createdAtTime, err := time.Parse(model.TimeFormat, createdAt)
	if err != nil {
		return nil, fmt.Errorf("解析活动创建时间失败: %w", err)
	}
	e.CreatedAt = createdAtTime
	updatedAtTime, err := time.Parse(model.TimeFormat, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("解析活动更新时间失败: %w", err)
	}
	e.UpdatedAt = updatedAtTime
	return e, nil
}

func (s *Store) UpdateEvent(id int64, req model.UpdateEventReq) (*model.Event, error) {
	event, err := s.GetEvent(id)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, model.ErrNotFound
	}

	if req.Title != nil {
		event.Title = *req.Title
	}
	if req.OrganizerID != nil {
		event.OrganizerID = *req.OrganizerID
	}
	if req.Description != nil {
		event.Description = *req.Description
	}
	if req.EventTime != nil {
		event.EventTime = *req.EventTime
	}
	if req.Location != nil {
		event.Location = *req.Location
	}
	if req.Capacity != nil {
		event.Capacity = *req.Capacity
	}
	if req.Price != nil {
		event.Price = *req.Price
	}
	if req.Status != nil {
		event.Status = *req.Status
	}

	now := time.Now().UTC().Format(model.TimeFormat)
	_, err = s.db.Exec(
		`UPDATE events SET organizer_id=?, title=?, description=?, event_time=?, location=?, capacity=?, price=?, status=?, updated_at=?
		 WHERE id=?`,
		event.OrganizerID, event.Title, event.Description, event.EventTime, event.Location,
		event.Capacity, event.Price, event.Status, now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("更新活动失败: %w", err)
	}

	return s.GetEvent(id)
}

func (s *Store) DeleteEvent(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM replies WHERE post_id IN (SELECT id FROM posts WHERE event_id = ?)`, id)
	if err != nil {
		return fmt.Errorf("删除回复失败: %w", err)
	}

	_, err = tx.Exec(`DELETE FROM posts WHERE event_id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除帖子失败: %w", err)
	}

	_, err = tx.Exec(`DELETE FROM tickets WHERE event_id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除门票失败: %w", err)
	}

	_, err = tx.Exec(`DELETE FROM registrations WHERE event_id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除报名记录失败: %w", err)
	}

	result, err := tx.Exec(`DELETE FROM events WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除活动失败: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取删除影响行数失败: %w", err)
	}
	if n == 0 {
		return model.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}
