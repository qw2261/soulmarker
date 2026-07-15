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

func (s *Store) ListMyActivities(userID int64, offset, limit int) ([]*model.MyActivity, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM admissions WHERE user_id = ?) +
		(SELECT COUNT(*) FROM registrations r
		 WHERE r.user_id = ?
		   AND NOT EXISTS (SELECT 1 FROM admissions a WHERE a.registration_id = r.id))`,
		userID, userID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询用户活动总数失败: %w", err)
	}

	query := `SELECT kind, source_id, registration_id, event_id, event_title, event_time,
		location, event_status, ticket_id, ticket_name, joined_at, admission_id,
		credential_code, admission_status, issued_at, revoked_at, checked_in_at
		FROM (
			SELECT 'admission' AS kind, a.id AS source_id, a.registration_id,
				a.event_id, e.title AS event_title, e.event_time, e.location,
				e.status AS event_status, r.ticket_id, a.ticket_name, a.issued_at AS joined_at,
				a.id AS admission_id, a.credential_code, a.status AS admission_status,
				a.issued_at, a.revoked_at, c.checked_in_at
			FROM admissions a
			JOIN events e ON e.id = a.event_id
			LEFT JOIN registrations r ON r.id = a.registration_id
			LEFT JOIN checkins c ON c.admission_id = a.id
			WHERE a.user_id = ?
			UNION ALL
			SELECT 'registration' AS kind, r.id AS source_id, r.id AS registration_id,
				r.event_id, e.title AS event_title, e.event_time, e.location,
				e.status AS event_status, r.ticket_id, r.ticket_name, r.created_at AS joined_at,
				NULL AS admission_id, NULL AS credential_code, NULL AS admission_status,
				NULL AS issued_at, NULL AS revoked_at, NULL AS checked_in_at
			FROM registrations r
			JOIN events e ON e.id = r.event_id
			WHERE r.user_id = ?
			  AND NOT EXISTS (SELECT 1 FROM admissions a WHERE a.registration_id = r.id)
		)
		ORDER BY datetime(event_time) DESC, datetime(joined_at) DESC, kind ASC, source_id DESC`
	args := []interface{}{userID, userID}
	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询用户活动列表失败: %w", err)
	}
	defer rows.Close()

	activities := make([]*model.MyActivity, 0)
	for rows.Next() {
		activity := &model.MyActivity{}
		var registrationID, ticketID, admissionID sql.NullInt64
		var joinedAt string
		var credentialCode, admissionStatus, issuedAt, revokedAt, checkedInAt sql.NullString
		if err := rows.Scan(
			&activity.Kind, &activity.ID, &registrationID, &activity.EventID,
			&activity.EventTitle, &activity.EventTime, &activity.Location, &activity.EventStatus,
			&ticketID, &activity.TicketName, &joinedAt, &admissionID, &credentialCode,
			&admissionStatus, &issuedAt, &revokedAt, &checkedInAt,
		); err != nil {
			return nil, 0, fmt.Errorf("读取用户活动记录失败: %w", err)
		}
		if registrationID.Valid {
			value := registrationID.Int64
			activity.RegistrationID = &value
		}
		if ticketID.Valid {
			value := ticketID.Int64
			activity.TicketID = &value
		}
		activity.JoinedAt, err = time.Parse(model.TimeFormat, joinedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("解析用户参与时间失败: %w", err)
		}
		if admissionID.Valid {
			admission := &model.Admission{
				ID: admissionID.Int64, RegistrationID: activity.RegistrationID,
				EventID: activity.EventID, UserID: userID, TicketName: activity.TicketName,
				CredentialCode: credentialCode.String, Status: admissionStatus.String,
			}
			admission.IssuedAt, err = time.Parse(model.TimeFormat, issuedAt.String)
			if err != nil {
				return nil, 0, fmt.Errorf("解析活动凭证签发时间失败: %w", err)
			}
			if revokedAt.Valid {
				value, err := time.Parse(model.TimeFormat, revokedAt.String)
				if err != nil {
					return nil, 0, fmt.Errorf("解析活动凭证吊销时间失败: %w", err)
				}
				admission.RevokedAt = &value
			}
			if checkedInAt.Valid {
				value, err := time.Parse(model.TimeFormat, checkedInAt.String)
				if err != nil {
					return nil, 0, fmt.Errorf("解析活动凭证核销时间失败: %w", err)
				}
				admission.CheckedInAt = &value
			}
			activity.Admission = admission
		}
		activities = append(activities, activity)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历用户活动记录失败: %w", err)
	}
	return activities, total, nil
}

func (s *Store) CheckIn(eventID int64, credentialCode, actor string, checkedInAt time.Time) (*model.Checkin, bool, error) {
	event, err := s.GetEvent(eventID)
	if err != nil {
		return nil, false, err
	}
	if event == nil {
		return nil, false, model.ErrAdmissionNotFound
	}
	return s.CheckInForOrganization(event.OrganizationID, eventID, credentialCode, actor, checkedInAt)
}

func (s *Store) CheckInForOrganization(
	organizationID, eventID int64,
	credentialCode, actor string,
	checkedInAt time.Time,
) (*model.Checkin, bool, error) {
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
		`SELECT a.id, a.status FROM admissions a JOIN events e ON e.id = a.event_id
		 WHERE a.event_id = ? AND a.credential_code = ? AND e.organization_id = ?`,
		eventID, credentialCode, organizationID,
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
	return s.listCheckins(0, eventID, offset, limit)
}

func (s *Store) ListCheckinsForOrganization(organizationID, eventID int64, offset, limit int) ([]*model.Checkin, int, error) {
	return s.listCheckins(organizationID, eventID, offset, limit)
}

func (s *Store) listCheckins(organizationID, eventID int64, offset, limit int) ([]*model.Checkin, int, error) {
	where := ` WHERE c.event_id = ?`
	args := []interface{}{eventID}
	if organizationID > 0 {
		where += ` AND e.organization_id = ?`
		args = append(args, organizationID)
	}
	var total int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM checkins c JOIN events e ON e.id = c.event_id`+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询核销总数失败: %w", err)
	}
	query := `SELECT c.id, c.admission_id, c.event_id, a.credential_code, u.name, u.contact,
		c.checked_in_at, c.checked_in_by
		FROM checkins c JOIN admissions a ON a.id = c.admission_id
		JOIN users u ON u.id = a.user_id JOIN events e ON e.id = c.event_id` + where + ` ORDER BY c.checked_in_at DESC`
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
