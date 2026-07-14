package store

import (
	"database/sql"
	"strings"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func (s *Store) CreateOrganizer(o *model.Organizer) error {
	now := time.Now().Format(model.TimeFormat)
	result, err := s.db.Exec(
		"INSERT INTO organizers (name, description, contact, logo_url, address, website, tags, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		o.Name, o.Description, o.Contact, o.LogoURL, o.Address, o.Website, o.Tags, now, now,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	o.ID = id
	o.CreatedAt, _ = time.Parse(model.TimeFormat, now)
	o.UpdatedAt = o.CreatedAt
	return nil
}

func (s *Store) GetOrganizer(id int64) (*model.Organizer, error) {
	if id == 0 {
		return nil, nil
	}
	o := &model.Organizer{}
	var createdAt, updatedAt string
	err := s.db.QueryRow(
		"SELECT id, name, description, contact, logo_url, address, website, tags, created_at, updated_at FROM organizers WHERE id = ?", id,
	).Scan(&o.ID, &o.Name, &o.Description, &o.Contact, &o.LogoURL, &o.Address, &o.Website, &o.Tags, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	o.CreatedAt, _ = time.Parse(model.TimeFormat, createdAt)
	o.UpdatedAt, _ = time.Parse(model.TimeFormat, updatedAt)
	return o, nil
}

func (s *Store) ListOrganizers(offset, limit int) ([]*model.Organizer, int, error) {
	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM organizers WHERE id <> 0").Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(
		`SELECT o.id, o.name, o.description, o.contact, o.logo_url, o.address, o.website, o.tags, o.created_at, o.updated_at,
		 COUNT(e.id) as event_count
		 FROM organizers o LEFT JOIN events e ON o.id = e.organizer_id
		 WHERE o.id <> 0
		 GROUP BY o.id ORDER BY o.created_at DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var organizers []*model.Organizer
	for rows.Next() {
		o := &model.Organizer{}
		var createdAt, updatedAt string
		if err := rows.Scan(&o.ID, &o.Name, &o.Description, &o.Contact, &o.LogoURL, &o.Address, &o.Website, &o.Tags, &createdAt, &updatedAt, &o.EventCount); err != nil {
			return nil, 0, err
		}
		o.CreatedAt, _ = time.Parse(model.TimeFormat, createdAt)
		o.UpdatedAt, _ = time.Parse(model.TimeFormat, updatedAt)
		organizers = append(organizers, o)
	}
	if organizers == nil {
		organizers = []*model.Organizer{}
	}
	return organizers, total, nil
}

func (s *Store) UpdateOrganizer(id int64, req model.UpdateOrganizerReq) (*model.Organizer, error) {
	setClauses := []string{}
	args := []interface{}{}

	if req.Name != nil {
		setClauses = append(setClauses, "name = ?")
		args = append(args, *req.Name)
	}
	if req.Description != nil {
		setClauses = append(setClauses, "description = ?")
		args = append(args, *req.Description)
	}
	if req.Contact != nil {
		setClauses = append(setClauses, "contact = ?")
		args = append(args, *req.Contact)
	}
	if req.LogoURL != nil {
		setClauses = append(setClauses, "logo_url = ?")
		args = append(args, *req.LogoURL)
	}
	if req.Address != nil {
		setClauses = append(setClauses, "address = ?")
		args = append(args, *req.Address)
	}
	if req.Website != nil {
		setClauses = append(setClauses, "website = ?")
		args = append(args, *req.Website)
	}
	if req.Tags != nil {
		setClauses = append(setClauses, "tags = ?")
		args = append(args, *req.Tags)
	}

	if len(setClauses) == 0 {
		return s.GetOrganizer(id)
	}

	now := time.Now().Format(model.TimeFormat)
	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, now)
	args = append(args, id)

	query := "UPDATE organizers SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"
	result, err := s.db.Exec(query, args...)
	if err != nil {
		return nil, err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return nil, model.ErrOrganizerNotFound
	}
	return s.GetOrganizer(id)
}

func (s *Store) DeleteOrganizer(id int64) error {
	if id == 0 {
		return model.ErrOrganizerNotFound
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var count int
	if err := tx.QueryRow("SELECT COUNT(*) FROM events WHERE organizer_id = ?", id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		if _, err := tx.Exec("UPDATE events SET organizer_id = 0 WHERE organizer_id = ?", id); err != nil {
			return err
		}
	}

	result, err := tx.Exec("DELETE FROM organizers WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return model.ErrOrganizerNotFound
	}

	return tx.Commit()
}
