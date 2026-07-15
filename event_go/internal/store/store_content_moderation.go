package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

const contentReportSelect = `SELECT cr.id, cr.event_id, cr.post_id, cr.target_type, cr.target_id,
	cr.reporter_user_id, u.name, cr.category, cr.detail, cr.status, cr.created_at,
	cr.resolved_at, cr.resolved_by, cr.resolution_note,
	CASE WHEN cr.target_type = 'post' THEN p.author_name ELSE COALESCE(r.author_name, '') END,
	p.title,
	CASE WHEN cr.target_type = 'post' THEN p.content ELSE COALESCE(r.content, '') END,
	CASE WHEN cr.target_type = 'post' THEN p.moderation_status ELSE COALESCE(r.moderation_status, 'removed') END
 FROM content_reports cr
 JOIN users u ON u.id = cr.reporter_user_id
 JOIN posts p ON p.id = cr.post_id
 LEFT JOIN replies r ON cr.target_type = 'reply' AND r.id = cr.target_id`

func scanContentReport(scanner rowScanner) (*model.ContentReport, error) {
	report := &model.ContentReport{}
	var createdAt string
	var resolvedAt sql.NullString
	if err := scanner.Scan(
		&report.ID, &report.EventID, &report.PostID, &report.TargetType, &report.TargetID,
		&report.ReporterUserID, &report.ReporterName, &report.Category, &report.Detail, &report.Status,
		&createdAt, &resolvedAt, &report.ResolvedBy, &report.ResolutionNote,
		&report.TargetAuthorName, &report.TargetTitle, &report.TargetContent, &report.TargetModerationStatus,
	); err != nil {
		return nil, err
	}
	parsedCreatedAt, err := time.Parse(model.TimeFormat, createdAt)
	if err != nil {
		return nil, fmt.Errorf("解析举报创建时间失败: %w", err)
	}
	report.CreatedAt = parsedCreatedAt
	if resolvedAt.Valid {
		parsedResolvedAt, err := time.Parse(model.TimeFormat, resolvedAt.String)
		if err != nil {
			return nil, fmt.Errorf("解析举报处理时间失败: %w", err)
		}
		report.ResolvedAt = &parsedResolvedAt
	}
	return report, nil
}

func (s *Store) GetPostForModeration(postID int64) (*model.Post, error) {
	post := &model.Post{}
	var createdAt string
	var moderatedAt sql.NullString
	err := s.db.QueryRow(
		`SELECT id, event_id, user_id, author_name, author_contact, title, content, identity_status,
		 moderation_status, moderated_at, moderated_by, moderation_reason, created_at,
		 (SELECT COUNT(*) FROM replies WHERE post_id = posts.id AND moderation_status = 'visible')
		 FROM posts WHERE id = ?`, postID,
	).Scan(&post.ID, &post.EventID, &post.UserID, &post.AuthorName, &post.AuthorContact, &post.Title,
		&post.Content, &post.IdentityStatus, &post.ModerationStatus, &moderatedAt, &post.ModeratedBy,
		&post.ModerationReason, &createdAt, &post.ReplyCount)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询治理帖子失败: %w", err)
	}
	parsedCreatedAt, err := time.Parse(model.TimeFormat, createdAt)
	if err != nil {
		return nil, fmt.Errorf("解析帖子创建时间失败: %w", err)
	}
	post.CreatedAt = parsedCreatedAt
	if moderatedAt.Valid {
		parsed, err := time.Parse(model.TimeFormat, moderatedAt.String)
		if err != nil {
			return nil, fmt.Errorf("解析帖子治理时间失败: %w", err)
		}
		post.ModeratedAt = &parsed
	}
	return post, nil
}

func (s *Store) GetReplyForModeration(replyID int64) (*model.Reply, error) {
	reply := &model.Reply{}
	var createdAt string
	var moderatedAt sql.NullString
	err := s.db.QueryRow(
		`SELECT id, post_id, user_id, author_name, author_contact, content, identity_status,
		 moderation_status, moderated_at, moderated_by, moderation_reason, created_at
		 FROM replies WHERE id = ?`, replyID,
	).Scan(&reply.ID, &reply.PostID, &reply.UserID, &reply.AuthorName, &reply.AuthorContact,
		&reply.Content, &reply.IdentityStatus, &reply.ModerationStatus, &moderatedAt,
		&reply.ModeratedBy, &reply.ModerationReason, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询治理回复失败: %w", err)
	}
	parsedCreatedAt, err := time.Parse(model.TimeFormat, createdAt)
	if err != nil {
		return nil, fmt.Errorf("解析回复创建时间失败: %w", err)
	}
	reply.CreatedAt = parsedCreatedAt
	if moderatedAt.Valid {
		parsed, err := time.Parse(model.TimeFormat, moderatedAt.String)
		if err != nil {
			return nil, fmt.Errorf("解析回复治理时间失败: %w", err)
		}
		reply.ModeratedAt = &parsed
	}
	return reply, nil
}

func (s *Store) CreateContentReport(report *model.ContentReport) (bool, error) {
	createdAt := report.CreatedAt.UTC().Format(model.TimeFormat)
	result, err := s.db.Exec(
		`INSERT INTO content_reports
		 (event_id, post_id, target_type, target_id, reporter_user_id, category, detail, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 'open', ?)`,
		report.EventID, report.PostID, report.TargetType, report.TargetID, report.ReporterUserID,
		report.Category, report.Detail, createdAt,
	)
	if isUniqueConstraintError(err) {
		existing, lookupErr := s.getOpenContentReport(report.ReporterUserID, report.TargetType, report.TargetID)
		if lookupErr != nil {
			return false, lookupErr
		}
		if existing == nil {
			return false, fmt.Errorf("查询重复举报失败: %w", err)
		}
		*report = *existing
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("创建内容举报失败: %w", err)
	}
	report.ID, err = result.LastInsertId()
	if err != nil {
		return false, fmt.Errorf("获取举报 ID 失败: %w", err)
	}
	report.Status = model.ContentReportStatusOpen
	return true, nil
}

func (s *Store) getOpenContentReport(reporterUserID int64, targetType string, targetID int64) (*model.ContentReport, error) {
	report, err := scanContentReport(s.db.QueryRow(
		contentReportSelect+` WHERE cr.reporter_user_id = ? AND cr.target_type = ? AND cr.target_id = ? AND cr.status = 'open'`,
		reporterUserID, targetType, targetID,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询未处理举报失败: %w", err)
	}
	return report, nil
}

func (s *Store) GetContentReport(id int64) (*model.ContentReport, error) {
	report, err := scanContentReport(s.db.QueryRow(contentReportSelect+` WHERE cr.id = ?`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询内容举报失败: %w", err)
	}
	return report, nil
}

func (s *Store) ListContentReports(params model.ListContentReportsParams) ([]*model.ContentReport, int, error) {
	where := ` WHERE 1=1`
	args := make([]interface{}, 0, 4)
	if params.Status != "" {
		where += ` AND cr.status = ?`
		args = append(args, params.Status)
	}
	if params.TargetType != "" {
		where += ` AND cr.target_type = ?`
		args = append(args, params.TargetType)
	}
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM content_reports cr`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询举报总数失败: %w", err)
	}
	query := contentReportSelect + where + ` ORDER BY CASE cr.status WHEN 'open' THEN 0 ELSE 1 END, cr.created_at DESC`
	if params.Limit > 0 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, params.Limit, params.Offset)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询举报列表失败: %w", err)
	}
	defer rows.Close()
	reports := make([]*model.ContentReport, 0)
	for rows.Next() {
		report, err := scanContentReport(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("读取举报记录失败: %w", err)
		}
		reports = append(reports, report)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历举报记录失败: %w", err)
	}
	return reports, total, nil
}

func lookupModerationTargetTx(tx *sql.Tx, eventID, postID int64, targetType string, targetID int64) (string, error) {
	var status string
	var err error
	if targetType == model.ContentTargetPost {
		err = tx.QueryRow(`SELECT moderation_status FROM posts WHERE id = ? AND event_id = ?`, targetID, eventID).Scan(&status)
		if postID != targetID {
			return "", model.ErrContentTargetNotFound
		}
	} else {
		err = tx.QueryRow(
			`SELECT r.moderation_status FROM replies r JOIN posts p ON p.id = r.post_id
			 WHERE r.id = ? AND r.post_id = ? AND p.event_id = ?`, targetID, postID, eventID,
		).Scan(&status)
	}
	if err == sql.ErrNoRows {
		return "", model.ErrContentTargetNotFound
	}
	if err != nil {
		return "", err
	}
	return status, nil
}

func updateModerationTargetTx(tx *sql.Tx, targetType string, targetID int64, status, actor, reason, now string) error {
	table := "posts"
	if targetType == model.ContentTargetReply {
		table = "replies"
	}
	_, err := tx.Exec(
		`UPDATE `+table+` SET moderation_status = ?, moderated_at = ?, moderated_by = ?, moderation_reason = ? WHERE id = ?`,
		status, now, actor, reason, targetID,
	)
	return err
}

func insertModerationActionTx(tx *sql.Tx, action *model.ContentModerationAction) error {
	_, err := tx.Exec(
		`INSERT INTO content_moderation_actions
		 (report_id, event_id, post_id, target_type, target_id, action, actor, reason, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		action.ReportID, action.EventID, action.PostID, action.TargetType, action.TargetID,
		action.Action, action.Actor, action.Reason, action.CreatedAt.UTC().Format(model.TimeFormat),
	)
	return err
}

func (s *Store) ResolveContentReport(id int64, resolution, note, actor string, now time.Time) (*model.ContentReport, error) {
	s.moderationMu.Lock()
	defer s.moderationMu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("开启举报处理事务失败: %w", err)
	}
	defer tx.Rollback()

	var eventID, postID, targetID int64
	var targetType, status string
	if err := tx.QueryRow(
		`SELECT event_id, post_id, target_type, target_id, status FROM content_reports WHERE id = ?`, id,
	).Scan(&eventID, &postID, &targetType, &targetID, &status); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrContentReportNotFound
		}
		return nil, fmt.Errorf("读取举报处理目标失败: %w", err)
	}
	if status != model.ContentReportStatusOpen {
		return nil, model.ErrContentReportResolved
	}
	if _, err := lookupModerationTargetTx(tx, eventID, postID, targetType, targetID); err != nil {
		return nil, err
	}
	nowValue := now.UTC().Format(model.TimeFormat)
	action := model.ContentModerationAction{
		ReportID: &id, EventID: eventID, PostID: postID, TargetType: targetType,
		TargetID: targetID, Actor: actor, Reason: note, CreatedAt: now,
	}
	if resolution == model.ContentResolutionRemove {
		if err := updateModerationTargetTx(tx, targetType, targetID, model.ModerationStatusRemoved, actor, note, nowValue); err != nil {
			return nil, fmt.Errorf("移除被举报内容失败: %w", err)
		}
		if _, err := tx.Exec(
			`UPDATE content_reports SET status = 'resolved', resolved_at = ?, resolved_by = ?, resolution_note = ?
			 WHERE target_type = ? AND target_id = ? AND status = 'open'`,
			nowValue, actor, note, targetType, targetID,
		); err != nil {
			return nil, fmt.Errorf("关闭关联举报失败: %w", err)
		}
		action.Action = model.ContentModerationActionRemove
	} else {
		if _, err := tx.Exec(
			`UPDATE content_reports SET status = 'dismissed', resolved_at = ?, resolved_by = ?, resolution_note = ?
			 WHERE id = ?`, nowValue, actor, note, id,
		); err != nil {
			return nil, fmt.Errorf("驳回举报失败: %w", err)
		}
		action.Action = model.ContentModerationActionDismiss
	}
	if err := insertModerationActionTx(tx, &action); err != nil {
		return nil, fmt.Errorf("记录举报处理动作失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交举报处理事务失败: %w", err)
	}
	return s.GetContentReport(id)
}

func (s *Store) ModerateContent(eventID, postID int64, targetType string, targetID int64, remove bool, actor, reason string, now time.Time) error {
	s.moderationMu.Lock()
	defer s.moderationMu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开启内容治理事务失败: %w", err)
	}
	defer tx.Rollback()
	currentStatus, err := lookupModerationTargetTx(tx, eventID, postID, targetType, targetID)
	if err != nil {
		return err
	}
	targetStatus := model.ModerationStatusVisible
	actionName := model.ContentModerationActionRestore
	if remove {
		if currentStatus == model.ModerationStatusRemoved {
			return model.ErrContentAlreadyRemoved
		}
		targetStatus = model.ModerationStatusRemoved
		actionName = model.ContentModerationActionRemove
	} else if currentStatus == model.ModerationStatusVisible {
		return model.ErrContentAlreadyVisible
	}
	nowValue := now.UTC().Format(model.TimeFormat)
	if err := updateModerationTargetTx(tx, targetType, targetID, targetStatus, actor, reason, nowValue); err != nil {
		return fmt.Errorf("更新内容治理状态失败: %w", err)
	}
	if remove {
		if _, err := tx.Exec(
			`UPDATE content_reports SET status = 'resolved', resolved_at = ?, resolved_by = ?, resolution_note = ?
			 WHERE target_type = ? AND target_id = ? AND status = 'open'`,
			nowValue, actor, reason, targetType, targetID,
		); err != nil {
			return fmt.Errorf("关闭内容举报失败: %w", err)
		}
	}
	action := model.ContentModerationAction{
		EventID: eventID, PostID: postID, TargetType: targetType, TargetID: targetID,
		Action: actionName, Actor: actor, Reason: reason, CreatedAt: now,
	}
	if err := insertModerationActionTx(tx, &action); err != nil {
		return fmt.Errorf("记录内容治理动作失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交内容治理事务失败: %w", err)
	}
	return nil
}

func (s *Store) ListContentModerationActions(offset, limit int) ([]*model.ContentModerationAction, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM content_moderation_actions`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询治理动作总数失败: %w", err)
	}
	query := `SELECT id, report_id, event_id, post_id, target_type, target_id, action, actor, reason, created_at
	 FROM content_moderation_actions ORDER BY created_at DESC, id DESC`
	args := []interface{}{}
	if limit > 0 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询治理动作失败: %w", err)
	}
	defer rows.Close()
	actions := make([]*model.ContentModerationAction, 0)
	for rows.Next() {
		action := &model.ContentModerationAction{}
		var reportID sql.NullInt64
		var createdAt string
		if err := rows.Scan(&action.ID, &reportID, &action.EventID, &action.PostID, &action.TargetType,
			&action.TargetID, &action.Action, &action.Actor, &action.Reason, &createdAt); err != nil {
			return nil, 0, fmt.Errorf("读取治理动作失败: %w", err)
		}
		if reportID.Valid {
			value := reportID.Int64
			action.ReportID = &value
		}
		parsed, err := time.Parse(model.TimeFormat, createdAt)
		if err != nil {
			return nil, 0, fmt.Errorf("解析治理动作时间失败: %w", err)
		}
		action.CreatedAt = parsed
		actions = append(actions, action)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历治理动作失败: %w", err)
	}
	return actions, total, nil
}
