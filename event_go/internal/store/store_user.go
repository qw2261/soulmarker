package store

import (
	"database/sql"
	"strings"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func (s *Store) CreateUser(u *model.User) error {
	now := time.Now().Format(model.TimeFormat)
	result, err := s.db.Exec(
		"INSERT INTO users (name, contact, password_hash, created_at) VALUES (?, ?, ?, ?)",
		u.Name, u.Contact, u.PasswordHash, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return model.ErrUserExists
		}
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	u.ID = id
	u.CreatedAt, _ = time.Parse(model.TimeFormat, now)
	return nil
}

func (s *Store) GetUserByContact(contact string) (*model.User, error) {
	row := s.db.QueryRow("SELECT id, name, contact, password_hash, created_at FROM users WHERE contact = ?", contact)
	u := &model.User{}
	var createdAt string
	err := row.Scan(&u.ID, &u.Name, &u.Contact, &u.PasswordHash, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.CreatedAt, _ = time.Parse(model.TimeFormat, createdAt)
	return u, nil
}

func (s *Store) GetUserByID(id int64) (*model.User, error) {
	row := s.db.QueryRow("SELECT id, name, contact, password_hash, created_at FROM users WHERE id = ?", id)
	u := &model.User{}
	var createdAt string
	err := row.Scan(&u.ID, &u.Name, &u.Contact, &u.PasswordHash, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.CreatedAt, _ = time.Parse(model.TimeFormat, createdAt)
	return u, nil
}
