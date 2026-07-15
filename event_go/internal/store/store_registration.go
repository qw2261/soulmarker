package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func (s *Store) Register(r *model.Registration) error {
	s.registrationMu.Lock()
	defer s.registrationMu.Unlock()

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
	effectivePrice := event.Price

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
		var price float64
		var name string
		err := tx.QueryRow(
			`SELECT name, price, stock FROM tickets WHERE id = ? AND event_id = ?`,
			*r.TicketID, r.EventID,
		).Scan(&name, &price, &stock)
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
		effectivePrice = price
	}

	now := time.Now().UTC().Format(model.TimeFormat)
	identityStatus := model.IdentityStatusLegacy
	if r.UserID != nil {
		identityStatus = model.IdentityStatusVerified
	}
	result, err := tx.Exec(
		`INSERT INTO registrations (event_id, user_id, name, contact, ticket_id, ticket_name, identity_status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		r.EventID, r.UserID, r.Name, r.Contact, r.TicketID, ticketName, identityStatus, now,
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

	var admission *model.Admission
	if effectivePrice == 0 && r.UserID != nil && r.Admission != nil && r.Admission.CredentialCode != "" {
		issuedAt := r.Admission.IssuedAt
		if issuedAt.IsZero() {
			issuedAt = time.Now().UTC()
		}
		result, err := tx.Exec(
			`INSERT INTO admissions (registration_id, event_id, user_id, ticket_name, credential_code, status, issued_at)
			 VALUES (?, ?, ?, ?, ?, 'active', ?)`,
			id, r.EventID, *r.UserID, ticketName, r.Admission.CredentialCode, issuedAt.Format(model.TimeFormat),
		)
		if err != nil {
			return fmt.Errorf("创建入场凭证失败: %w", err)
		}
		admissionID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("获取入场凭证 ID 失败: %w", err)
		}
		registrationID := id
		admission = &model.Admission{
			ID: admissionID, RegistrationID: &registrationID, EventID: r.EventID,
			UserID: *r.UserID, TicketName: ticketName, CredentialCode: r.Admission.CredentialCode,
			Status: model.AdmissionStatusActive, IssuedAt: issuedAt,
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	r.ID = id
	r.TicketName = ticketName
	r.IdentityStatus = identityStatus
	createdAt, _ := time.Parse(model.TimeFormat, now)
	r.CreatedAt = createdAt
	r.Admission = admission
	return nil
}

func (s *Store) ListRegistrations(eventID int64, offset, limit int) ([]*model.Registration, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM registrations WHERE event_id = ?`, eventID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询报名总数失败: %w", err)
	}

	query := `SELECT id, event_id, user_id, name, contact, ticket_id, ticket_name, identity_status, created_at
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
		if err := rows.Scan(&r.ID, &r.EventID, &r.UserID, &r.Name, &r.Contact, &r.TicketID, &r.TicketName, &r.IdentityStatus, &createdAt); err != nil {
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

func (s *Store) IsRegisteredByUserID(eventID, userID int64) (bool, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM registrations WHERE event_id = ? AND user_id = ?`,
		eventID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("查询用户报名信息失败: %w", err)
	}
	return count > 0, nil
}

func (s *Store) CancelRegistration(eventID int64, contact string) error {
	s.registrationMu.Lock()
	defer s.registrationMu.Unlock()
	return s.cancelRegistration(eventID, "contact", contact)
}

func (s *Store) CancelRegistrationByUserID(eventID, userID int64) error {
	s.registrationMu.Lock()
	defer s.registrationMu.Unlock()
	return s.cancelRegistration(eventID, "user_id", userID)
}

func (s *Store) cancelRegistration(eventID int64, identityColumn string, identity interface{}) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	var registrationID int64
	var ticketID sql.NullInt64
	query := fmt.Sprintf(`SELECT id, ticket_id FROM registrations WHERE event_id = ? AND %s = ?`, identityColumn)
	err = tx.QueryRow(query, eventID, identity).Scan(&registrationID, &ticketID)
	if err == sql.ErrNoRows {
		return model.ErrNotRegistered
	}
	if err != nil {
		return fmt.Errorf("查询报名记录失败: %w", err)
	}

	var admissionID int64
	err = tx.QueryRow(`SELECT id FROM admissions WHERE registration_id = ?`, registrationID).Scan(&admissionID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("查询入场凭证失败: %w", err)
	}
	if err == nil {
		var checkinID int64
		checkinErr := tx.QueryRow(`SELECT id FROM checkins WHERE admission_id = ?`, admissionID).Scan(&checkinID)
		if checkinErr == nil {
			return model.ErrAdmissionCheckedIn
		}
		if checkinErr != sql.ErrNoRows {
			return fmt.Errorf("查询核销记录失败: %w", checkinErr)
		}
		if _, err := tx.Exec(
			`UPDATE admissions SET status = 'revoked', revoked_at = ? WHERE id = ? AND status = 'active'`,
			time.Now().UTC().Format(model.TimeFormat), admissionID,
		); err != nil {
			return fmt.Errorf("吊销入场凭证失败: %w", err)
		}
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

	deleteQuery := fmt.Sprintf(`DELETE FROM registrations WHERE event_id = ? AND %s = ?`, identityColumn)
	_, err = tx.Exec(deleteQuery, eventID, identity)
	if err != nil {
		return fmt.Errorf("删除报名记录失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}

func (s *Store) ListMyRegistrations(userID int64, offset, limit int) ([]*model.MyRegistration, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM registrations WHERE user_id = ?`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询用户报名总数失败: %w", err)
	}

	query := `SELECT r.id, r.event_id, e.title, e.event_time, e.location, e.status,
		 r.ticket_id, r.ticket_name, r.created_at
		 FROM registrations r
		 JOIN events e ON e.id = r.event_id
		 WHERE r.user_id = ?
		 ORDER BY e.event_time DESC, r.created_at DESC`
	args := []interface{}{userID}
	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询用户报名列表失败: %w", err)
	}
	defer rows.Close()

	registrations := make([]*model.MyRegistration, 0)
	for rows.Next() {
		registration := &model.MyRegistration{}
		var createdAt string
		if err := rows.Scan(
			&registration.ID,
			&registration.EventID,
			&registration.EventTitle,
			&registration.EventTime,
			&registration.Location,
			&registration.EventStatus,
			&registration.TicketID,
			&registration.TicketName,
			&createdAt,
		); err != nil {
			return nil, 0, fmt.Errorf("读取用户报名记录失败: %w", err)
		}
		createdAtTime, err := time.Parse(model.TimeFormat, createdAt)
		if err != nil {
			return nil, 0, fmt.Errorf("解析用户报名时间失败: %w", err)
		}
		registration.CreatedAt = createdAtTime
		registrations = append(registrations, registration)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历用户报名记录失败: %w", err)
	}
	return registrations, total, nil
}
