package store

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

type Store struct {
	db *sql.DB
}

func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Store) Ping() error {
	if s.db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	return s.db.Ping()
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

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Printf("⚠️ 设置 WAL 模式失败: %v", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		log.Printf("⚠️ 设置 busy_timeout 失败: %v", err)
	}

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
		`CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			author_name TEXT NOT NULL,
			author_contact TEXT NOT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY (event_id) REFERENCES events(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_event ON posts(event_id)`,
		`CREATE TABLE IF NOT EXISTS replies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id INTEGER NOT NULL,
			author_name TEXT NOT NULL,
			author_contact TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY (post_id) REFERENCES posts(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_replies_post ON replies(post_id)`,
		`CREATE TABLE IF NOT EXISTS tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			price REAL NOT NULL DEFAULT 0,
			stock INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (event_id) REFERENCES events(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_event ON tickets(event_id)`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			log.Fatalf("数据库迁移失败: %v", err)
		}
	}

	migrations := []string{
		`ALTER TABLE registrations ADD COLUMN ticket_id INTEGER REFERENCES tickets(id)`,
		`ALTER TABLE registrations ADD COLUMN ticket_name TEXT NOT NULL DEFAULT ''`,
	}
	for _, q := range migrations {
		s.db.Exec(q)
	}
}

func (s *Store) CreateEvent(e *model.Event) error {
	now := time.Now().UTC().Format(model.TimeFormat)
	result, err := s.db.Exec(
		`INSERT INTO events (title, description, event_time, location, capacity, price, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 'published', ?, ?)`,
		e.Title, e.Description, e.EventTime, e.Location, e.Capacity, e.Price, now, now,
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

func (s *Store) ListEvents(status string, priceType string, keyword string) ([]*model.Event, error) {
	query := `SELECT id, title, description, event_time, location, capacity, price, status, created_at, updated_at
			FROM events WHERE 1=1`
	var args []interface{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	if priceType == "free" {
		query += " AND price = 0"
	} else if priceType == "paid" {
		query += " AND price > 0"
	}

	if keyword != "" {
		query += " AND (title LIKE ? OR description LIKE ?)"
		args = append(args, "%"+keyword+"%", "%"+keyword+"%")
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询活动列表失败: %w", err)
	}
	defer rows.Close()

	var events []*model.Event
	for rows.Next() {
		e := &model.Event{}
		var createdAt, updatedAt string
		if err := rows.Scan(&e.ID, &e.Title, &e.Description, &e.EventTime, &e.Location,
			&e.Capacity, &e.Price, &e.Status, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("读取活动记录失败: %w", err)
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
		events = append(events, e)
	}

	if events == nil {
		events = []*model.Event{}
	}
	return events, nil
}

func (s *Store) GetEvent(id int64) (*model.Event, error) {
	e := &model.Event{}
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
		`UPDATE events SET title=?, description=?, event_time=?, location=?, capacity=?, price=?, status=?, updated_at=?
		 WHERE id=?`,
		event.Title, event.Description, event.EventTime, event.Location,
		event.Capacity, event.Price, event.Status, now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("更新活动失败: %w", err)
	}

	event.UpdatedAt, _ = time.Parse(model.TimeFormat, now)
	return event, nil
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

func (s *Store) ListRegistrations(eventID int64) ([]*model.Registration, error) {
	rows, err := s.db.Query(
		`SELECT id, event_id, name, contact, created_at
		 FROM registrations WHERE event_id = ? ORDER BY created_at ASC`, eventID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询报名列表失败: %w", err)
	}
	defer rows.Close()

	var registrations []*model.Registration
	for rows.Next() {
		r := &model.Registration{}
		var createdAt string
		if err := rows.Scan(&r.ID, &r.EventID, &r.Name, &r.Contact, &createdAt); err != nil {
			return nil, fmt.Errorf("读取报名记录失败: %w", err)
		}
		createdAtTime, err := time.Parse(model.TimeFormat, createdAt)
		if err != nil {
			return nil, fmt.Errorf("解析报名时间失败: %w", err)
		}
		r.CreatedAt = createdAtTime
		registrations = append(registrations, r)
	}

	if registrations == nil {
		registrations = []*model.Registration{}
	}
	return registrations, nil
}

func isUniqueConstraintError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE")
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

func (s *Store) CreatePost(p *model.Post) error {
	now := time.Now().UTC().Format(model.TimeFormat)
	result, err := s.db.Exec(
		`INSERT INTO posts (event_id, author_name, author_contact, title, content, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		p.EventID, p.AuthorName, p.AuthorContact, p.Title, p.Content, now,
	)
	if err != nil {
		return fmt.Errorf("创建帖子失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取帖子 ID 失败: %w", err)
	}
	p.ID = id
	p.ReplyCount = 0
	createdAt, _ := time.Parse(model.TimeFormat, now)
	p.CreatedAt = createdAt
	return nil
}

func (s *Store) ListPosts(eventID int64) ([]*model.Post, error) {
	rows, err := s.db.Query(
		`SELECT p.id, p.event_id, p.author_name, p.title, p.content, p.created_at,
		        (SELECT COUNT(*) FROM replies WHERE post_id = p.id) AS reply_count
		 FROM posts p WHERE p.event_id = ? ORDER BY p.created_at DESC`, eventID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询帖子列表失败: %w", err)
	}
	defer rows.Close()

	var posts []*model.Post
	for rows.Next() {
		p := &model.Post{}
		var createdAt string
		if err := rows.Scan(&p.ID, &p.EventID, &p.AuthorName, &p.Title, &p.Content,
			&createdAt, &p.ReplyCount); err != nil {
			return nil, fmt.Errorf("读取帖子记录失败: %w", err)
		}
		createdAtTime, err := time.Parse(model.TimeFormat, createdAt)
		if err != nil {
			return nil, fmt.Errorf("解析帖子创建时间失败: %w", err)
		}
		p.CreatedAt = createdAtTime
		posts = append(posts, p)
	}

	if posts == nil {
		posts = []*model.Post{}
	}
	return posts, nil
}

func (s *Store) GetPost(postID int64) (*model.Post, error) {
	p := &model.Post{}
	var createdAt string
	err := s.db.QueryRow(
		`SELECT p.id, p.event_id, p.author_name, p.title, p.content, p.created_at,
		        (SELECT COUNT(*) FROM replies WHERE post_id = p.id) AS reply_count
		 FROM posts p WHERE p.id = ?`, postID,
	).Scan(&p.ID, &p.EventID, &p.AuthorName, &p.Title, &p.Content, &createdAt, &p.ReplyCount)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询帖子失败: %w", err)
	}
	createdAtTime, err := time.Parse(model.TimeFormat, createdAt)
	if err != nil {
		return nil, fmt.Errorf("解析帖子创建时间失败: %w", err)
	}
	p.CreatedAt = createdAtTime
	return p, nil
}

func (s *Store) CreateReply(r *model.Reply) error {
	now := time.Now().UTC().Format(model.TimeFormat)
	result, err := s.db.Exec(
		`INSERT INTO replies (post_id, author_name, author_contact, content, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		r.PostID, r.AuthorName, r.AuthorContact, r.Content, now,
	)
	if err != nil {
		return fmt.Errorf("创建回复失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取回复 ID 失败: %w", err)
	}
	r.ID = id
	createdAt, _ := time.Parse(model.TimeFormat, now)
	r.CreatedAt = createdAt
	return nil
}

func (s *Store) ListReplies(postID int64) ([]*model.Reply, error) {
	rows, err := s.db.Query(
		`SELECT id, post_id, author_name, content, created_at
		 FROM replies WHERE post_id = ? ORDER BY created_at ASC`, postID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询回复列表失败: %w", err)
	}
	defer rows.Close()

	var replies []*model.Reply
	for rows.Next() {
		r := &model.Reply{}
		var createdAt string
		if err := rows.Scan(&r.ID, &r.PostID, &r.AuthorName, &r.Content, &createdAt); err != nil {
			return nil, fmt.Errorf("读取回复记录失败: %w", err)
		}
		createdAtTime, err := time.Parse(model.TimeFormat, createdAt)
		if err != nil {
			return nil, fmt.Errorf("解析回复创建时间失败: %w", err)
		}
		r.CreatedAt = createdAtTime
		replies = append(replies, r)
	}

	if replies == nil {
		replies = []*model.Reply{}
	}
	return replies, nil
}

func (s *Store) CreateTicket(t *model.Ticket) error {
	now := time.Now().UTC().Format(model.TimeFormat)
	result, err := s.db.Exec(
		`INSERT INTO tickets (event_id, name, price, stock, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		t.EventID, t.Name, t.Price, t.Stock, now, now,
	)
	if err != nil {
		return fmt.Errorf("创建门票失败: %w", err)
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

func (s *Store) ListTickets(eventID int64) ([]*model.Ticket, error) {
	rows, err := s.db.Query(
		`SELECT id, event_id, name, price, stock, created_at, updated_at
		 FROM tickets WHERE event_id = ? ORDER BY created_at ASC`, eventID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询门票列表失败: %w", err)
	}
	defer rows.Close()

	var tickets []*model.Ticket
	for rows.Next() {
		t := &model.Ticket{}
		var createdAt, updatedAt string
		if err := rows.Scan(&t.ID, &t.EventID, &t.Name, &t.Price, &t.Stock, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("读取门票记录失败: %w", err)
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
		tickets = append(tickets, t)
	}

	if tickets == nil {
		tickets = []*model.Ticket{}
	}
	return tickets, nil
}

func (s *Store) GetTicket(ticketID int64) (*model.Ticket, error) {
	t := &model.Ticket{}
	var createdAt, updatedAt string
	err := s.db.QueryRow(
		`SELECT id, event_id, name, price, stock, created_at, updated_at
		 FROM tickets WHERE id = ?`, ticketID,
	).Scan(&t.ID, &t.EventID, &t.Name, &t.Price, &t.Stock, &createdAt, &updatedAt)
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
	_, err = s.db.Exec(
		`UPDATE tickets SET name=?, price=?, stock=?, updated_at=? WHERE id=?`,
		ticket.Name, ticket.Price, ticket.Stock, now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("更新门票失败: %w", err)
	}

	ticket.UpdatedAt, _ = time.Parse(model.TimeFormat, now)
	return ticket, nil
}

func (s *Store) DeleteTicket(id int64) error {
	result, err := s.db.Exec(`DELETE FROM tickets WHERE id = ?`, id)
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
	return nil
}
