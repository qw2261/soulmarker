package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) *Store {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("创建数据库目录失败: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA busy_timeout=5000")

	store := &Store{db: db}
	store.migrate()
	return store
}

func (s *Store) migrate() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			event_time TEXT NOT NULL,
			location TEXT NOT NULL,
			capacity INTEGER NOT NULL DEFAULT 0,
			price REAL NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'published',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS registrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			contact TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY (event_id) REFERENCES events(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_registrations_event ON registrations(event_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_registrations_event_contact 
		 ON registrations(event_id, contact)`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			log.Fatalf("数据库迁移失败: %v", err)
		}
	}
}

func (s *Store) CreateEvent(e *Event) error {
	now := time.Now().UTC().Format(timeFormat)
	result, err := s.db.Exec(
		`INSERT INTO events (title, description, event_time, location, capacity, price, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 'published', ?, ?)`,
		e.Title, e.Description, e.EventTime, e.Location, e.Capacity, e.Price, now, now,
	)
	if err != nil {
		return fmt.Errorf("创建活动失败: %w", err)
	}

	id, _ := result.LastInsertId()
	e.ID = id
	e.Status = "published"
	e.CreatedAt, _ = time.Parse(timeFormat, now)
	e.UpdatedAt = e.CreatedAt
	return nil
}

func (s *Store) ListEvents() ([]*Event, error) {
	rows, err := s.db.Query(
		`SELECT id, title, description, event_time, location, capacity, price, status, created_at, updated_at
		 FROM events ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("查询活动列表失败: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		e := &Event{}
		var createdAt, updatedAt string
		if err := rows.Scan(&e.ID, &e.Title, &e.Description, &e.EventTime, &e.Location,
			&e.Capacity, &e.Price, &e.Status, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("读取活动记录失败: %w", err)
		}
		e.CreatedAt, _ = time.Parse(timeFormat, createdAt)
		e.UpdatedAt, _ = time.Parse(timeFormat, updatedAt)
		events = append(events, e)
	}

	if events == nil {
		events = []*Event{}
	}
	return events, nil
}

func (s *Store) GetEvent(id int64) (*Event, error) {
	e := &Event{}
	var createdAt, updatedAt string
	err := s.db.QueryRow(
		`SELECT id, title, description, event_time, location, capacity, price, status, created_at, updated_at
		 FROM events WHERE id = ?`, id,
	).Scan(&e.ID, &e.Title, &e.Description, &e.EventTime, &e.Location,
		&e.Capacity, &e.Price, &e.Status, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询活动失败: %w", err)
	}
	e.CreatedAt, _ = time.Parse(timeFormat, createdAt)
	e.UpdatedAt, _ = time.Parse(timeFormat, updatedAt)
	return e, nil
}

func (s *Store) UpdateEvent(id int64, req UpdateEventReq) (*Event, error) {
	event, err := s.GetEvent(id)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, ErrNotFound
	}

	if req.Title != nil {
		event.Title = *req.Title
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

	now := time.Now().UTC().Format(timeFormat)
	_, err = s.db.Exec(
		`UPDATE events SET title=?, description=?, event_time=?, location=?, capacity=?, price=?, status=?, updated_at=?
		 WHERE id=?`,
		event.Title, event.Description, event.EventTime, event.Location,
		event.Capacity, event.Price, event.Status, now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("更新活动失败: %w", err)
	}

	event.UpdatedAt, _ = time.Parse(timeFormat, now)
	return event, nil
}

func (s *Store) DeleteEvent(id int64) error {
	_, err := s.db.Exec(`DELETE FROM registrations WHERE event_id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除报名记录失败: %w", err)
	}

	result, err := s.db.Exec(`DELETE FROM events WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除活动失败: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Register(r *Registration) error {
	event, err := s.GetEvent(r.EventID)
	if err != nil {
		return err
	}
	if event == nil {
		return ErrNotFound
	}

	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM registrations WHERE event_id = ?`, r.EventID).Scan(&count)
	if count >= event.Capacity {
		return ErrFull
	}

	now := time.Now().UTC().Format(timeFormat)
	_, err = s.db.Exec(
		`INSERT INTO registrations (event_id, name, contact, created_at) VALUES (?, ?, ?, ?)`,
		r.EventID, r.Name, r.Contact, now,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrDuplicate
		}
		return fmt.Errorf("报名失败: %w", err)
	}

	r.CreatedAt, _ = time.Parse(timeFormat, now)
	return nil
}

func (s *Store) ListRegistrations(eventID int64) ([]*Registration, error) {
	rows, err := s.db.Query(
		`SELECT id, event_id, name, contact, created_at
		 FROM registrations WHERE event_id = ? ORDER BY created_at ASC`, eventID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询报名列表失败: %w", err)
	}
	defer rows.Close()

	var registrations []*Registration
	for rows.Next() {
		r := &Registration{}
		var createdAt string
		if err := rows.Scan(&r.ID, &r.EventID, &r.Name, &r.Contact, &createdAt); err != nil {
			return nil, fmt.Errorf("读取报名记录失败: %w", err)
		}
		r.CreatedAt, _ = time.Parse(timeFormat, createdAt)
		registrations = append(registrations, r)
	}

	if registrations == nil {
		registrations = []*Registration{}
	}
	return registrations, nil
}

func isUniqueConstraintError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE")
}
