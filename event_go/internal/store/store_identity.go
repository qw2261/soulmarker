package store

import (
	"fmt"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func (s *Store) GetIdentityMigrationReport(legacyLimit int) (*model.IdentityMigrationReport, error) {
	if legacyLimit <= 0 || legacyLimit > 100 {
		legacyLimit = 100
	}

	registrations, err := s.identityEntityStats("registrations")
	if err != nil {
		return nil, err
	}
	posts, err := s.identityEntityStats("posts")
	if err != nil {
		return nil, err
	}
	replies, err := s.identityEntityStats("replies")
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(`
		SELECT entity_type, id, parent_id, name, contact FROM (
			SELECT 'registration' AS entity_type, id, event_id AS parent_id, name, contact
			FROM registrations WHERE identity_status = 'legacy'
			UNION ALL
			SELECT 'post' AS entity_type, id, event_id AS parent_id, author_name AS name, author_contact AS contact
			FROM posts WHERE identity_status = 'legacy'
			UNION ALL
			SELECT 'reply' AS entity_type, id, post_id AS parent_id, author_name AS name, author_contact AS contact
			FROM replies WHERE identity_status = 'legacy'
		)
		ORDER BY entity_type, id
		LIMIT ?`, legacyLimit)
	if err != nil {
		return nil, fmt.Errorf("查询 legacy 身份记录失败: %w", err)
	}
	defer rows.Close()

	legacyRecords := make([]model.LegacyIdentityRecord, 0)
	for rows.Next() {
		var record model.LegacyIdentityRecord
		if err := rows.Scan(&record.EntityType, &record.ID, &record.ParentID, &record.Name, &record.Contact); err != nil {
			return nil, fmt.Errorf("读取 legacy 身份记录失败: %w", err)
		}
		legacyRecords = append(legacyRecords, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 legacy 身份记录失败: %w", err)
	}

	return &model.IdentityMigrationReport{
		Registrations: registrations,
		Posts:         posts,
		Replies:       replies,
		LegacyRecords: legacyRecords,
	}, nil
}

func (s *Store) identityEntityStats(table string) (model.IdentityEntityStats, error) {
	var stats model.IdentityEntityStats
	query := fmt.Sprintf(`SELECT COUNT(*),
		COALESCE(SUM(CASE WHEN identity_status = 'verified' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN identity_status = 'backfilled' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN identity_status = 'legacy' THEN 1 ELSE 0 END), 0)
		FROM %s`, table)
	if err := s.db.QueryRow(query).Scan(&stats.Total, &stats.Verified, &stats.Backfilled, &stats.Legacy); err != nil {
		return model.IdentityEntityStats{}, fmt.Errorf("统计 %s 身份迁移结果失败: %w", table, err)
	}
	return stats, nil
}
