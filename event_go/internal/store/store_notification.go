package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func insertNotificationTx(tx *sql.Tx, notification *model.Notification) (bool, error) {
	var eventID interface{}
	if notification.EventID != nil {
		eventID = *notification.EventID
	}
	result, err := tx.Exec(
		`INSERT INTO notifications
		 (user_id, event_id, type, title, body, action_url, idempotency_key, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(idempotency_key) DO NOTHING`,
		notification.UserID, eventID, notification.Type, notification.Title, notification.Body,
		notification.ActionURL, notification.IdempotencyKey,
		notification.CreatedAt.UTC().Format(model.TimeFormat),
	)
	if err != nil {
		return false, err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if inserted == 0 {
		return false, nil
	}
	notification.ID, err = result.LastInsertId()
	return true, err
}

func scanNotification(scanner rowScanner) (*model.Notification, error) {
	notification := &model.Notification{}
	var eventID sql.NullInt64
	var readAt sql.NullString
	var createdAt string
	if err := scanner.Scan(
		&notification.ID, &notification.UserID, &eventID, &notification.Type,
		&notification.Title, &notification.Body, &notification.ActionURL,
		&readAt, &createdAt,
	); err != nil {
		return nil, err
	}
	if eventID.Valid {
		value := eventID.Int64
		notification.EventID = &value
	}
	if readAt.Valid {
		parsed, err := time.Parse(model.TimeFormat, readAt.String)
		if err != nil {
			return nil, fmt.Errorf("解析通知已读时间失败: %w", err)
		}
		notification.ReadAt = &parsed
	}
	parsed, err := time.Parse(model.TimeFormat, createdAt)
	if err != nil {
		return nil, fmt.Errorf("解析通知创建时间失败: %w", err)
	}
	notification.CreatedAt = parsed
	return notification, nil
}

func (s *Store) ListNotifications(params model.ListNotificationsParams) ([]*model.Notification, int, error) {
	where := ` WHERE user_id = ?`
	args := []interface{}{params.UserID}
	if params.UnreadOnly {
		where += ` AND read_at IS NULL`
	}
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM notifications`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询通知总数失败: %w", err)
	}
	query := `SELECT id, user_id, event_id, type, title, body, action_url, read_at, created_at
		FROM notifications` + where + ` ORDER BY created_at DESC, id DESC`
	if params.Limit > 0 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, params.Limit, params.Offset)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询通知列表失败: %w", err)
	}
	defer rows.Close()
	notifications := make([]*model.Notification, 0)
	for rows.Next() {
		notification, err := scanNotification(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("读取通知失败: %w", err)
		}
		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历通知失败: %w", err)
	}
	return notifications, total, nil
}

func (s *Store) CountUnreadNotifications(userID int64) (int, error) {
	var total int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read_at IS NULL`, userID,
	).Scan(&total); err != nil {
		return 0, fmt.Errorf("查询未读通知数失败: %w", err)
	}
	return total, nil
}

func (s *Store) MarkNotificationRead(userID, notificationID int64, readAt time.Time) error {
	result, err := s.db.Exec(
		`UPDATE notifications SET read_at = COALESCE(read_at, ?) WHERE id = ? AND user_id = ?`,
		readAt.UTC().Format(model.TimeFormat), notificationID, userID,
	)
	if err != nil {
		return fmt.Errorf("标记通知已读失败: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("读取通知已读结果失败: %w", err)
	}
	if updated == 0 {
		return model.ErrNotificationNotFound
	}
	return nil
}

func (s *Store) MarkAllNotificationsRead(userID int64, readAt time.Time) (int, error) {
	result, err := s.db.Exec(
		`UPDATE notifications SET read_at = ? WHERE user_id = ? AND read_at IS NULL`,
		readAt.UTC().Format(model.TimeFormat), userID,
	)
	if err != nil {
		return 0, fmt.Errorf("全部标记通知已读失败: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("读取全部已读结果失败: %w", err)
	}
	return int(updated), nil
}

func (s *Store) CreateDueEventReminders(now time.Time, window time.Duration) (int, error) {
	s.notificationMu.Lock()
	defer s.notificationMu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("开启活动提醒事务失败: %w", err)
	}
	defer tx.Rollback()
	type reminderCandidate struct {
		userID    int64
		eventID   int64
		title     string
		eventTime string
		location  string
	}
	rows, err := tx.Query(
		`SELECT r.user_id, e.id, e.title, e.event_time, e.location
		 FROM registrations r JOIN events e ON e.id = r.event_id
		 WHERE r.user_id IS NOT NULL AND e.status = 'published'`,
	)
	if err != nil {
		return 0, fmt.Errorf("查询活动提醒候选失败: %w", err)
	}
	candidates := make([]reminderCandidate, 0)
	for rows.Next() {
		candidate := reminderCandidate{}
		if err := rows.Scan(&candidate.userID, &candidate.eventID, &candidate.title, &candidate.eventTime, &candidate.location); err != nil {
			rows.Close()
			return 0, fmt.Errorf("读取活动提醒候选失败: %w", err)
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("遍历活动提醒候选失败: %w", err)
	}
	if err := rows.Close(); err != nil {
		return 0, fmt.Errorf("关闭活动提醒候选查询失败: %w", err)
	}
	created := 0
	for _, candidate := range candidates {
		eventTime, err := time.Parse(model.TimeFormat, candidate.eventTime)
		if err != nil {
			return 0, fmt.Errorf("解析活动提醒时间失败: %w", err)
		}
		if !eventTime.After(now) || eventTime.After(now.Add(window)) {
			continue
		}
		eventID := candidate.eventID
		notification := &model.Notification{
			UserID: candidate.userID, EventID: &eventID,
			Type: model.NotificationEventReminder24H, Title: "活动即将开始",
			Body:           fmt.Sprintf("活动“%s”将在 %s 开始，地点：%s。", candidate.title, eventTime.Format("2006-01-02 15:04 MST"), candidate.location),
			ActionURL:      fmt.Sprintf("/events/%d", candidate.eventID),
			IdempotencyKey: fmt.Sprintf("event-reminder-24h:%d:%d:%s", candidate.eventID, candidate.userID, candidate.eventTime),
			CreatedAt:      now,
		}
		inserted, err := insertNotificationTx(tx, notification)
		if err != nil {
			return 0, fmt.Errorf("创建活动提醒失败: %w", err)
		}
		if inserted {
			created++
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("提交活动提醒事务失败: %w", err)
	}
	return created, nil
}
