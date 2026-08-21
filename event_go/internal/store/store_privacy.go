package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

const dataSubjectRequestColumns = `id, user_id, request_type, status, requested_at, processed_at, processed_by, resolution, created_at`

func validDataSubjectRequestType(t string) bool {
	return t == model.DataSubjectRequestAccountErasure || t == model.DataSubjectRequestDataExport
}

func validDataSubjectStatus(s string) bool {
	return s == model.DataSubjectStatusPending ||
		s == model.DataSubjectStatusCompleted ||
		s == model.DataSubjectStatusRejected ||
		s == model.DataSubjectStatusFailed
}

func scanDataSubjectRequest(row rowScanner) (*model.DataSubjectRequest, error) {
	req := &model.DataSubjectRequest{}
	var requestedAt, createdAt string
	var processedAt sql.NullString
	var processedBy sql.NullInt64
	err := row.Scan(
		&req.ID, &req.UserID, &req.RequestType, &req.Status,
		&requestedAt, &processedAt, &processedBy, &req.Resolution, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	req.RequestedAt, _ = time.Parse(model.TimeFormat, requestedAt)
	req.CreatedAt, _ = time.Parse(model.TimeFormat, createdAt)
	if processedAt.Valid {
		value, err := time.Parse(model.TimeFormat, processedAt.String)
		if err != nil {
			return nil, fmt.Errorf("解析数据主体请求处理时间失败: %w", err)
		}
		req.ProcessedAt = &value
	}
	if processedBy.Valid {
		value := processedBy.Int64
		req.ProcessedBy = &value
	}
	return req, nil
}

// CreateDataSubjectRequest 记录一条新的数据主体请求，初始状态为 pending。
func (s *Store) CreateDataSubjectRequest(userID int64, requestType string) (*model.DataSubjectRequest, error) {
	if !validDataSubjectRequestType(requestType) {
		return nil, fmt.Errorf("无效的数据主体请求类型: %s", requestType)
	}
	now := time.Now().UTC()
	result, err := s.db.Exec(
		`INSERT INTO data_subject_requests (user_id, request_type, status, requested_at, resolution, created_at)
		 VALUES (?, ?, ?, ?, '', ?)`,
		userID, requestType, model.DataSubjectStatusPending,
		now.Format(model.TimeFormat), now.Format(model.TimeFormat),
	)
	if err != nil {
		return nil, fmt.Errorf("创建数据主体请求失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("读取数据主体请求 ID 失败: %w", err)
	}
	return &model.DataSubjectRequest{
		ID:          id,
		UserID:      userID,
		RequestType: requestType,
		Status:      model.DataSubjectStatusPending,
		RequestedAt: now,
		CreatedAt:   now,
	}, nil
}

// ListDataSubjectRequests 列出指定用户的数据主体请求，按请求时间倒序。
func (s *Store) ListDataSubjectRequests(userID int64) ([]*model.DataSubjectRequest, error) {
	rows, err := s.db.Query(
		`SELECT `+dataSubjectRequestColumns+`
		 FROM data_subject_requests
		 WHERE user_id = ?
		 ORDER BY requested_at DESC, id DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("列出数据主体请求失败: %w", err)
	}
	defer rows.Close()
	var requests []*model.DataSubjectRequest
	for rows.Next() {
		request, err := scanDataSubjectRequest(rows)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历数据主体请求失败: %w", err)
	}
	return requests, nil
}

// CompleteDataSubjectRequest 将一条 pending 请求标记为已处理。
func (s *Store) CompleteDataSubjectRequest(id, userID int64, status, resolution string, processedBy int64) error {
	if status == model.DataSubjectStatusPending || !validDataSubjectStatus(status) {
		return fmt.Errorf("无效的数据主体请求终态: %s", status)
	}
	processedAt := time.Now().UTC().Format(model.TimeFormat)
	result, err := s.db.Exec(
		`UPDATE data_subject_requests
		 SET status = ?, processed_at = ?, processed_by = ?, resolution = ?
		 WHERE id = ? AND user_id = ? AND status = ?`,
		status, processedAt, processedBy, strings.TrimSpace(resolution),
		id, userID, model.DataSubjectStatusPending,
	)
	if err != nil {
		return fmt.Errorf("更新数据主体请求状态失败: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("读取数据主体请求更新结果失败: %w", err)
	}
	if affected == 0 {
		// 区分请求不存在与已被处理，避免误报。
		exists, err := s.dataSubjectRequestExists(id, userID)
		if err != nil {
			return err
		}
		if !exists {
			return model.ErrDataSubjectRequestNotFound
		}
		return model.ErrDataSubjectRequestNotPending
	}
	return nil
}

func (s *Store) dataSubjectRequestExists(id, userID int64) (bool, error) {
	var exists int
	err := s.db.QueryRow(
		`SELECT 1 FROM data_subject_requests WHERE id = ? AND user_id = ?`, id, userID,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("查询数据主体请求失败: %w", err)
	}
	return true, nil
}

// ExportUserData 聚合导出该用户在系统中的全部个人数据，满足数据可携带权。
func (s *Store) ExportUserData(userID int64) (*model.UserDataExport, error) {
	export := &model.UserDataExport{UserID: userID, ExportedAt: time.Now().UTC()}

	profile, err := s.exportUserProfile(userID)
	if err != nil {
		return nil, err
	}
	export.Profile = profile

	if export.Memberships, err = s.exportUserMemberships(userID); err != nil {
		return nil, err
	}
	if export.Registrations, err = s.exportUserRegistrations(userID); err != nil {
		return nil, err
	}
	if export.AuthoredPosts, err = s.exportUserPosts(userID); err != nil {
		return nil, err
	}
	if export.AuthoredReplies, err = s.exportUserReplies(userID); err != nil {
		return nil, err
	}
	if export.Notifications, err = s.exportUserNotifications(userID); err != nil {
		return nil, err
	}
	privacyRequests, err := s.ListDataSubjectRequests(userID)
	if err != nil {
		return nil, err
	}
	export.PrivacyRequests = make([]model.DataSubjectRequest, 0, len(privacyRequests))
	for _, request := range privacyRequests {
		export.PrivacyRequests = append(export.PrivacyRequests, *request)
	}
	return export, nil
}

func (s *Store) exportUserProfile(userID int64) (*model.UserDataExportProfile, error) {
	var createdAt string
	profile := &model.UserDataExportProfile{}
	err := s.db.QueryRow(
		`SELECT name, contact, recovery_email, created_at FROM users WHERE id = ?`, userID,
	).Scan(&profile.Name, &profile.Contact, &profile.RecoveryEmail, &createdAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("用户不存在")
	}
	if err != nil {
		return nil, fmt.Errorf("查询用户档案失败: %w", err)
	}
	profile.CreatedAt, _ = time.Parse(model.TimeFormat, createdAt)
	return profile, nil
}

func (s *Store) exportUserMemberships(userID int64) ([]model.UserDataExportMembership, error) {
	rows, err := s.db.Query(
		`SELECT o.name, o.slug, m.role, m.status, m.created_at
		 FROM organization_members m
		 JOIN organizations o ON o.id = m.organization_id
		 WHERE m.user_id = ?
		 ORDER BY m.created_at ASC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询组织成员关系失败: %w", err)
	}
	defer rows.Close()
	var memberships []model.UserDataExportMembership
	for rows.Next() {
		var m model.UserDataExportMembership
		var createdAt string
		if err := rows.Scan(&m.OrganizationName, &m.OrganizationSlug, &m.Role, &m.Status, &createdAt); err != nil {
			return nil, fmt.Errorf("读取组织成员关系失败: %w", err)
		}
		m.CreatedAt, _ = time.Parse(model.TimeFormat, createdAt)
		memberships = append(memberships, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历组织成员关系失败: %w", err)
	}
	return memberships, nil
}

func (s *Store) exportUserRegistrations(userID int64) ([]model.UserDataExportRegistration, error) {
	rows, err := s.db.Query(
		`SELECT r.event_id, COALESCE(e.title, ''), r.name, r.contact, r.created_at
		 FROM registrations r
		 LEFT JOIN events e ON e.id = r.event_id
		 WHERE r.user_id = ?
		 ORDER BY r.created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询用户报名记录失败: %w", err)
	}
	defer rows.Close()
	var registrations []model.UserDataExportRegistration
	for rows.Next() {
		var reg model.UserDataExportRegistration
		var createdAt string
		if err := rows.Scan(&reg.EventID, &reg.EventTitle, &reg.Name, &reg.Contact, &createdAt); err != nil {
			return nil, fmt.Errorf("读取用户报名记录失败: %w", err)
		}
		reg.CreatedAt, _ = time.Parse(model.TimeFormat, createdAt)
		registrations = append(registrations, reg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历用户报名记录失败: %w", err)
	}
	return registrations, nil
}

func (s *Store) exportUserPosts(userID int64) ([]model.UserDataExportPost, error) {
	rows, err := s.db.Query(
		`SELECT event_id, title, content, created_at FROM posts
		 WHERE user_id = ?
		 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询用户发帖失败: %w", err)
	}
	defer rows.Close()
	var posts []model.UserDataExportPost
	for rows.Next() {
		var post model.UserDataExportPost
		var createdAt string
		if err := rows.Scan(&post.EventID, &post.Title, &post.Content, &createdAt); err != nil {
			return nil, fmt.Errorf("读取用户发帖失败: %w", err)
		}
		post.CreatedAt, _ = time.Parse(model.TimeFormat, createdAt)
		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历用户发帖失败: %w", err)
	}
	return posts, nil
}

func (s *Store) exportUserReplies(userID int64) ([]model.UserDataExportReply, error) {
	rows, err := s.db.Query(
		`SELECT p.event_id, r.content, r.created_at
		 FROM replies r
		 JOIN posts p ON p.id = r.post_id
		 WHERE r.user_id = ?
		 ORDER BY r.created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询用户回复失败: %w", err)
	}
	defer rows.Close()
	var replies []model.UserDataExportReply
	for rows.Next() {
		var reply model.UserDataExportReply
		var createdAt string
		if err := rows.Scan(&reply.EventID, &reply.Content, &createdAt); err != nil {
			return nil, fmt.Errorf("读取用户回复失败: %w", err)
		}
		reply.CreatedAt, _ = time.Parse(model.TimeFormat, createdAt)
		replies = append(replies, reply)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历用户回复失败: %w", err)
	}
	return replies, nil
}

func (s *Store) exportUserNotifications(userID int64) ([]model.UserDataExportNotification, error) {
	rows, err := s.db.Query(
		`SELECT type, title, body, read_at, created_at FROM notifications
		 WHERE user_id = ?
		 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询用户通知失败: %w", err)
	}
	defer rows.Close()
	var notifications []model.UserDataExportNotification
	for rows.Next() {
		var n model.UserDataExportNotification
		var readAt sql.NullString
		var createdAt string
		if err := rows.Scan(&n.Type, &n.Title, &n.Body, &readAt, &createdAt); err != nil {
			return nil, fmt.Errorf("读取用户通知失败: %w", err)
		}
		n.CreatedAt, _ = time.Parse(model.TimeFormat, createdAt)
		if readAt.Valid {
			value, err := time.Parse(model.TimeFormat, readAt.String)
			if err != nil {
				return nil, fmt.Errorf("解析通知已读时间失败: %w", err)
			}
			n.ReadAt = &value
		}
		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历用户通知失败: %w", err)
	}
	return notifications, nil
}

// ErasureUser 在一个事务内完成账号注销：标记 deleted_at、递增 auth_version 撤销会话、
// 撤销未使用的恢复/重置令牌，并将账号注销请求置为 completed。
func (s *Store) ErasureUser(userID int64) (*model.DataSubjectRequest, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("开启账号注销事务失败: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	timestamp := now.Format(model.TimeFormat)

	request, err := createDataSubjectRequestTx(tx, userID, model.DataSubjectRequestAccountErasure, timestamp)
	if err != nil {
		return nil, err
	}

	result, err := tx.Exec(
		`UPDATE users SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`, timestamp, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("标记用户注销失败: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("读取用户注销结果失败: %w", err)
	}
	if affected == 0 {
		return nil, model.ErrUserAlreadyDeleted
	}

	if _, err := tx.Exec(
		`UPDATE user_auth_versions SET version = version + 1, updated_at = ? WHERE user_id = ?`,
		timestamp, userID,
	); err != nil {
		return nil, fmt.Errorf("撤销用户会话失败: %w", err)
	}
	if _, err := tx.Exec(
		`UPDATE password_reset_tokens SET used_at = ? WHERE user_id = ? AND used_at IS NULL`,
		timestamp, userID,
	); err != nil {
		return nil, fmt.Errorf("撤销密码重置令牌失败: %w", err)
	}
	if _, err := tx.Exec(
		`UPDATE recovery_email_tokens SET used_at = ? WHERE user_id = ? AND used_at IS NULL`,
		timestamp, userID,
	); err != nil {
		return nil, fmt.Errorf("撤销恢复邮箱令牌失败: %w", err)
	}
	if _, err := tx.Exec(
		`UPDATE data_subject_requests
		 SET status = ?, processed_at = ?, processed_by = ?, resolution = ?
		 WHERE id = ? AND user_id = ?`,
		model.DataSubjectStatusCompleted, timestamp, userID, "账号已注销并撤销全部会话", request.ID, userID,
	); err != nil {
		return nil, fmt.Errorf("更新账号注销请求状态失败: %w", err)
	}

	// 同步返回对象的处置状态，避免向客户端暴露 pending 的错误终态。
	request.Status = model.DataSubjectStatusCompleted
	request.ProcessedAt = &now
	request.ProcessedBy = &userID
	request.Resolution = "账号已注销并撤销全部会话"

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交账号注销事务失败: %w", err)
	}
	return request, nil
}

func createDataSubjectRequestTx(tx *sql.Tx, userID int64, requestType, timestamp string) (*model.DataSubjectRequest, error) {
	result, err := tx.Exec(
		`INSERT INTO data_subject_requests (user_id, request_type, status, requested_at, resolution, created_at)
		 VALUES (?, ?, ?, ?, '', ?)`,
		userID, requestType, model.DataSubjectStatusPending, timestamp, timestamp,
	)
	if err != nil {
		return nil, fmt.Errorf("创建账号注销请求失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("读取账号注销请求 ID 失败: %w", err)
	}
	parsed, _ := time.Parse(model.TimeFormat, timestamp)
	return &model.DataSubjectRequest{
		ID: id, UserID: userID, RequestType: requestType,
		Status: model.DataSubjectStatusPending, RequestedAt: parsed, CreatedAt: parsed,
	}, nil
}

// SweepExpiredRequests 清理超过留存期的已处理数据主体请求，返回清理数量，用于数据留存流程。
// pending 请求不会被清理，以保留用户未完成请求的可追溯性。
func (s *Store) SweepExpiredRequests(retainedBefore time.Time) (int, error) {
	result, err := s.db.Exec(
		`DELETE FROM data_subject_requests
		 WHERE status <> ?
		   AND processed_at IS NOT NULL
		   AND processed_at < ?`,
		model.DataSubjectStatusPending, retainedBefore.UTC().Format(model.TimeFormat),
	)
	if err != nil {
		return 0, fmt.Errorf("清理过期数据主体请求失败: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("读取清理结果失败: %w", err)
	}
	return int(affected), nil
}
