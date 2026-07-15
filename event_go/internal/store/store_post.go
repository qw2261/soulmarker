package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func (s *Store) CreatePost(p *model.Post) error {
	now := time.Now().UTC().Format(model.TimeFormat)
	identityStatus := model.IdentityStatusLegacy
	if p.UserID != nil {
		identityStatus = model.IdentityStatusVerified
	}
	result, err := s.db.Exec(
		`INSERT INTO posts (event_id, user_id, author_name, author_contact, title, content, identity_status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.EventID, p.UserID, p.AuthorName, p.AuthorContact, p.Title, p.Content, identityStatus, now,
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
	p.IdentityStatus = identityStatus
	p.ModerationStatus = model.ModerationStatusVisible
	createdAt, _ := time.Parse(model.TimeFormat, now)
	p.CreatedAt = createdAt
	return nil
}

func (s *Store) ListPosts(eventID int64, offset, limit int) ([]*model.Post, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM posts WHERE event_id = ? AND moderation_status = 'visible'`, eventID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询帖子总数失败: %w", err)
	}

	query := `SELECT p.id, p.event_id, p.user_id, p.author_name, p.title, p.content, p.identity_status, p.created_at,
		        (SELECT COUNT(*) FROM replies WHERE post_id = p.id AND moderation_status = 'visible') AS reply_count
		 FROM posts p WHERE p.event_id = ? AND p.moderation_status = 'visible' ORDER BY p.created_at DESC`
	args := []interface{}{eventID}
	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询帖子列表失败: %w", err)
	}
	defer rows.Close()

	var posts []*model.Post
	for rows.Next() {
		p := &model.Post{}
		var createdAt string
		if err := rows.Scan(&p.ID, &p.EventID, &p.UserID, &p.AuthorName, &p.Title, &p.Content,
			&p.IdentityStatus, &createdAt, &p.ReplyCount); err != nil {
			return nil, 0, fmt.Errorf("读取帖子记录失败: %w", err)
		}
		createdAtTime, err := time.Parse(model.TimeFormat, createdAt)
		if err != nil {
			return nil, 0, fmt.Errorf("解析帖子创建时间失败: %w", err)
		}
		p.CreatedAt = createdAtTime
		posts = append(posts, p)
	}

	if posts == nil {
		posts = []*model.Post{}
	}
	return posts, total, nil
}

func (s *Store) GetPost(postID int64) (*model.Post, error) {
	p := &model.Post{}
	var createdAt string
	err := s.db.QueryRow(
		`SELECT p.id, p.event_id, p.user_id, p.author_name, p.title, p.content, p.identity_status, p.created_at,
		        (SELECT COUNT(*) FROM replies WHERE post_id = p.id AND moderation_status = 'visible') AS reply_count
		 FROM posts p WHERE p.id = ? AND p.moderation_status = 'visible'`, postID,
	).Scan(&p.ID, &p.EventID, &p.UserID, &p.AuthorName, &p.Title, &p.Content, &p.IdentityStatus, &createdAt, &p.ReplyCount)
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
	identityStatus := model.IdentityStatusLegacy
	if r.UserID != nil {
		identityStatus = model.IdentityStatusVerified
	}
	result, err := s.db.Exec(
		`INSERT INTO replies (post_id, user_id, author_name, author_contact, content, identity_status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.PostID, r.UserID, r.AuthorName, r.AuthorContact, r.Content, identityStatus, now,
	)
	if err != nil {
		return fmt.Errorf("创建回复失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取回复 ID 失败: %w", err)
	}
	r.ID = id
	r.IdentityStatus = identityStatus
	r.ModerationStatus = model.ModerationStatusVisible
	createdAt, _ := time.Parse(model.TimeFormat, now)
	r.CreatedAt = createdAt
	return nil
}

func (s *Store) ListReplies(postID int64) ([]*model.Reply, error) {
	rows, err := s.db.Query(
		`SELECT id, post_id, user_id, author_name, content, identity_status, created_at
		 FROM replies WHERE post_id = ? AND moderation_status = 'visible' ORDER BY created_at ASC`, postID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询回复列表失败: %w", err)
	}
	defer rows.Close()

	var replies []*model.Reply
	for rows.Next() {
		r := &model.Reply{}
		var createdAt string
		if err := rows.Scan(&r.ID, &r.PostID, &r.UserID, &r.AuthorName, &r.Content, &r.IdentityStatus, &createdAt); err != nil {
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
