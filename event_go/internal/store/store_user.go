package store

import (
	"database/sql"
	"fmt"
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
	u.AuthVersion = 1
	u.CreatedAt, _ = time.Parse(model.TimeFormat, now)
	return nil
}

func (s *Store) GetUserByContact(contact string) (*model.User, error) {
	row := s.db.QueryRow(`SELECT u.id, u.name, u.contact, u.password_hash,
		COALESCE(v.version, 1), u.created_at
		FROM users u LEFT JOIN user_auth_versions v ON v.user_id = u.id
		WHERE u.contact = ?`, contact)
	u := &model.User{}
	var createdAt string
	err := row.Scan(&u.ID, &u.Name, &u.Contact, &u.PasswordHash, &u.AuthVersion, &createdAt)
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
	row := s.db.QueryRow(`SELECT u.id, u.name, u.contact, u.password_hash,
		COALESCE(v.version, 1), u.created_at
		FROM users u LEFT JOIN user_auth_versions v ON v.user_id = u.id
		WHERE u.id = ?`, id)
	u := &model.User{}
	var createdAt string
	err := row.Scan(&u.ID, &u.Name, &u.Contact, &u.PasswordHash, &u.AuthVersion, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.CreatedAt, _ = time.Parse(model.TimeFormat, createdAt)
	return u, nil
}

func (s *Store) CreatePasswordResetToken(userID int64, tokenHash string, createdAt, expiresAt time.Time) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开启密码重置事务失败: %w", err)
	}
	defer tx.Rollback()
	timestamp := createdAt.UTC().Format(model.TimeFormat)
	var latestCreatedAt string
	err = tx.QueryRow(
		`SELECT created_at FROM password_reset_tokens WHERE user_id = ? ORDER BY created_at DESC LIMIT 1`,
		userID,
	).Scan(&latestCreatedAt)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("查询最近密码重置请求失败: %w", err)
	}
	if err == nil {
		latest, parseErr := time.Parse(model.TimeFormat, latestCreatedAt)
		if parseErr != nil {
			return fmt.Errorf("解析最近密码重置时间失败: %w", parseErr)
		}
		if createdAt.UTC().Before(latest.Add(time.Minute)) {
			return model.ErrPasswordResetRateLimit
		}
	}
	if _, err := tx.Exec(
		`UPDATE password_reset_tokens SET used_at = ? WHERE user_id = ? AND used_at IS NULL`,
		timestamp, userID,
	); err != nil {
		return fmt.Errorf("撤销旧密码重置令牌失败: %w", err)
	}
	if _, err := tx.Exec(
		`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at, created_at)
		 VALUES (?, ?, ?, ?)`,
		userID, tokenHash, expiresAt.UTC().Format(model.TimeFormat), timestamp,
	); err != nil {
		return fmt.Errorf("创建密码重置令牌失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交密码重置事务失败: %w", err)
	}
	return nil
}

func (s *Store) ResetPassword(tokenHash, passwordHash string, resetAt time.Time) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开启密码更新事务失败: %w", err)
	}
	defer tx.Rollback()
	timestamp := resetAt.UTC().Format(model.TimeFormat)
	var userID int64
	err = tx.QueryRow(
		`SELECT user_id FROM password_reset_tokens
		 WHERE token_hash = ? AND used_at IS NULL AND expires_at > ?`,
		tokenHash, timestamp,
	).Scan(&userID)
	if err == sql.ErrNoRows {
		return model.ErrPasswordResetInvalid
	}
	if err != nil {
		return fmt.Errorf("查询密码重置令牌失败: %w", err)
	}
	if _, err := tx.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, userID); err != nil {
		return fmt.Errorf("更新用户密码失败: %w", err)
	}
	if _, err := tx.Exec(
		`UPDATE user_auth_versions SET version = version + 1, updated_at = ? WHERE user_id = ?`,
		timestamp, userID,
	); err != nil {
		return fmt.Errorf("撤销用户会话失败: %w", err)
	}
	if _, err := tx.Exec(
		`UPDATE password_reset_tokens SET used_at = ? WHERE user_id = ? AND used_at IS NULL`,
		timestamp, userID,
	); err != nil {
		return fmt.Errorf("消费密码重置令牌失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交密码更新事务失败: %w", err)
	}
	return nil
}

func (s *Store) RevokeUserSessions(userID int64, revokedAt time.Time) error {
	result, err := s.db.Exec(
		`UPDATE user_auth_versions SET version = version + 1, updated_at = ? WHERE user_id = ?`,
		revokedAt.UTC().Format(model.TimeFormat), userID,
	)
	if err != nil {
		return fmt.Errorf("撤销用户会话失败: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("读取会话撤销结果失败: %w", err)
	}
	if updated == 0 {
		return model.ErrInvalidCreds
	}
	return nil
}
