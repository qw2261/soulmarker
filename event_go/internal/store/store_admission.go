package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanMyAdmission(scanner rowScanner) (*model.MyAdmission, error) {
	admission := &model.MyAdmission{}
	var registrationID sql.NullInt64
	var issuedAt string
	var revokedAt, checkedInAt sql.NullString
	err := scanner.Scan(
		&admission.ID, &registrationID, &admission.EventID, &admission.UserID,
		&admission.TicketName, &admission.CredentialCode, &admission.Status,
		&issuedAt, &revokedAt, &checkedInAt, &admission.EventTitle,
		&admission.EventTime, &admission.Location, &admission.EventStatus,
	)
	if err != nil {
		return nil, err
	}
	if registrationID.Valid {
		value := registrationID.Int64
		admission.RegistrationID = &value
	}
	parsedIssuedAt, err := time.Parse(model.TimeFormat, issuedAt)
	if err != nil {
		return nil, fmt.Errorf("解析凭证签发时间失败: %w", err)
	}
	admission.IssuedAt = parsedIssuedAt
	if revokedAt.Valid {
		value, err := time.Parse(model.TimeFormat, revokedAt.String)
		if err != nil {
			return nil, fmt.Errorf("解析凭证吊销时间失败: %w", err)
		}
		admission.RevokedAt = &value
	}
	if checkedInAt.Valid {
		value, err := time.Parse(model.TimeFormat, checkedInAt.String)
		if err != nil {
			return nil, fmt.Errorf("解析凭证核销时间失败: %w", err)
		}
		admission.CheckedInAt = &value
	}
	return admission, nil
}

const myAdmissionSelect = `SELECT a.id, a.registration_id, a.event_id, a.user_id,
	 a.ticket_name, a.credential_code, a.status, a.issued_at, a.revoked_at,
	 c.checked_in_at, e.title, e.event_time, e.location, e.status
	 FROM admissions a
	 JOIN events e ON e.id = a.event_id
	 LEFT JOIN checkins c ON c.admission_id = a.id`

func (s *Store) GetAdmissionByUser(eventID, userID int64) (*model.MyAdmission, error) {
	admission, err := scanMyAdmission(s.db.QueryRow(
		myAdmissionSelect+` WHERE a.event_id = ? AND a.user_id = ? ORDER BY a.issued_at DESC LIMIT 1`,
		eventID, userID,
	))
	if err == sql.ErrNoRows {
		return nil, model.ErrAdmissionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询用户入场凭证失败: %w", err)
	}
	return admission, nil
}

func (s *Store) ListMyAdmissions(userID int64, offset, limit int) ([]*model.MyAdmission, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM admissions WHERE user_id = ?`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询用户入场凭证总数失败: %w", err)
	}
	query := myAdmissionSelect + ` WHERE a.user_id = ? ORDER BY e.event_time DESC, a.issued_at DESC`
	args := []interface{}{userID}
	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询用户入场凭证失败: %w", err)
	}
	defer rows.Close()
	admissions := make([]*model.MyAdmission, 0)
	for rows.Next() {
		admission, err := scanMyAdmission(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("读取用户入场凭证失败: %w", err)
		}
		admissions = append(admissions, admission)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历用户入场凭证失败: %w", err)
	}
	return admissions, total, nil
}

func (s *Store) CheckIn(eventID int64, credentialCode, actor string, checkedInAt time.Time) (*model.Checkin, bool, error) {
	s.checkinMu.Lock()
	defer s.checkinMu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return nil, false, fmt.Errorf("开启核销事务失败: %w", err)
	}
	defer tx.Rollback()

	var admissionID int64
	var status string
	err = tx.QueryRow(
		`SELECT id, status FROM admissions WHERE event_id = ? AND credential_code = ?`,
		eventID, credentialCode,
	).Scan(&admissionID, &status)
	if err == sql.ErrNoRows {
		return nil, false, model.ErrAdmissionNotFound
	}
	if err != nil {
		return nil, false, fmt.Errorf("查询待核销凭证失败: %w", err)
	}
	if status != model.AdmissionStatusActive {
		return nil, false, model.ErrAdmissionRevoked
	}

	checkin, err := getCheckinTx(tx, admissionID)
	if err == nil {
		return checkin, true, nil
	}
	if err != sql.ErrNoRows {
		return nil, false, err
	}

	timestamp := checkedInAt.UTC().Format(model.TimeFormat)
	_, err = tx.Exec(
		`INSERT INTO checkins (admission_id, event_id, checked_in_at, checked_in_by) VALUES (?, ?, ?, ?)`,
		admissionID, eventID, timestamp, actor,
	)
	if err != nil {
		return nil, false, fmt.Errorf("写入核销记录失败: %w", err)
	}
	checkin, err = getCheckinTx(tx, admissionID)
	if err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("提交核销事务失败: %w", err)
	}
	return checkin, false, nil
}

func getCheckinTx(tx *sql.Tx, admissionID int64) (*model.Checkin, error) {
	checkin := &model.Checkin{}
	var checkedInAt string
	err := tx.QueryRow(
		`SELECT c.id, c.admission_id, c.event_id, a.credential_code, u.name, u.contact,
		 c.checked_in_at, c.checked_in_by
		 FROM checkins c
		 JOIN admissions a ON a.id = c.admission_id
		 JOIN users u ON u.id = a.user_id
		 WHERE c.admission_id = ?`,
		admissionID,
	).Scan(
		&checkin.ID, &checkin.AdmissionID, &checkin.EventID, &checkin.CredentialCode,
		&checkin.UserName, &checkin.UserContact, &checkedInAt, &checkin.CheckedInBy,
	)
	if err != nil {
		return nil, err
	}
	parsed, err := time.Parse(model.TimeFormat, checkedInAt)
	if err != nil {
		return nil, fmt.Errorf("解析核销时间失败: %w", err)
	}
	checkin.CheckedInAt = parsed
	return checkin, nil
}

func (s *Store) ListCheckins(eventID int64, offset, limit int) ([]*model.Checkin, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM checkins WHERE event_id = ?`, eventID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询核销总数失败: %w", err)
	}
	query := `SELECT c.id, c.admission_id, c.event_id, a.credential_code, u.name, u.contact,
		c.checked_in_at, c.checked_in_by
		FROM checkins c JOIN admissions a ON a.id = c.admission_id
		JOIN users u ON u.id = a.user_id WHERE c.event_id = ? ORDER BY c.checked_in_at DESC`
	args := []interface{}{eventID}
	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询核销记录失败: %w", err)
	}
	defer rows.Close()
	checkins := make([]*model.Checkin, 0)
	for rows.Next() {
		checkin := &model.Checkin{}
		var checkedInAt string
		if err := rows.Scan(
			&checkin.ID, &checkin.AdmissionID, &checkin.EventID, &checkin.CredentialCode,
			&checkin.UserName, &checkin.UserContact, &checkedInAt, &checkin.CheckedInBy,
		); err != nil {
			return nil, 0, fmt.Errorf("读取核销记录失败: %w", err)
		}
		parsed, err := time.Parse(model.TimeFormat, checkedInAt)
		if err != nil {
			return nil, 0, fmt.Errorf("解析核销时间失败: %w", err)
		}
		checkin.CheckedInAt = parsed
		checkins = append(checkins, checkin)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历核销记录失败: %w", err)
	}
	return checkins, total, nil
}
