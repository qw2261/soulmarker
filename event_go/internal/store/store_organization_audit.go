package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func validAuditActorType(value string) bool {
	switch value {
	case model.AuditActorOrganizationMember, model.AuditActorPlatformAdmin, model.AuditActorSystem:
		return true
	default:
		return false
	}
}

func validAuditOutcome(value string) bool {
	switch value {
	case model.AuditOutcomeSuccess, model.AuditOutcomeDenied, model.AuditOutcomeFailure:
		return true
	default:
		return false
	}
}

func (s *Store) AppendOrganizationAudit(entry *model.OrganizationAuditLog) error {
	if entry == nil || entry.OrganizationID <= 0 || !validAuditActorType(entry.ActorType) ||
		!validAuditOutcome(entry.Outcome) || strings.TrimSpace(entry.Action) == "" ||
		strings.TrimSpace(entry.ResourceType) == "" || strings.TrimSpace(entry.RequestID) == "" ||
		entry.HTTPStatus < 100 || entry.HTTPStatus > 599 {
		return fmt.Errorf("invalid organization audit entry")
	}
	createdAt := entry.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	result, err := s.db.Exec(
		`INSERT INTO organization_audit_logs
		 (organization_id, actor_type, actor_id, action, resource_type, resource_id,
		  request_id, outcome, http_status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.OrganizationID, entry.ActorType, entry.ActorID, strings.TrimSpace(entry.Action),
		strings.TrimSpace(entry.ResourceType), strings.TrimSpace(entry.ResourceID),
		strings.TrimSpace(entry.RequestID), entry.Outcome, entry.HTTPStatus,
		createdAt.Format(model.TimeFormat),
	)
	if err != nil {
		return fmt.Errorf("append organization audit: %w", err)
	}
	entry.ID, err = result.LastInsertId()
	if err != nil {
		return fmt.Errorf("read organization audit id: %w", err)
	}
	entry.CreatedAt = createdAt
	return nil
}

func (s *Store) ListOrganizationAudits(organizationID int64, offset, limit int) ([]*model.OrganizationAuditLog, int, error) {
	if organizationID <= 0 {
		return nil, 0, fmt.Errorf("organization scope is required")
	}
	var total int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM organization_audit_logs WHERE organization_id = ?`, organizationID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count organization audits: %w", err)
	}
	query := `SELECT id, organization_id, actor_type, actor_id, action, resource_type,
		 resource_id, request_id, outcome, http_status, created_at
		 FROM organization_audit_logs WHERE organization_id = ?
		 ORDER BY created_at DESC, id DESC`
	args := []any{organizationID}
	if limit > 0 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list organization audits: %w", err)
	}
	defer rows.Close()
	entries := make([]*model.OrganizationAuditLog, 0)
	for rows.Next() {
		entry := &model.OrganizationAuditLog{}
		var actorID sql.NullInt64
		var createdAt string
		if err := rows.Scan(
			&entry.ID, &entry.OrganizationID, &entry.ActorType, &actorID, &entry.Action,
			&entry.ResourceType, &entry.ResourceID, &entry.RequestID, &entry.Outcome,
			&entry.HTTPStatus, &createdAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan organization audit: %w", err)
		}
		if actorID.Valid {
			value := actorID.Int64
			entry.ActorID = &value
		}
		entry.CreatedAt, err = time.Parse(model.TimeFormat, createdAt)
		if err != nil {
			return nil, 0, fmt.Errorf("parse organization audit time: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate organization audits: %w", err)
	}
	return entries, total, nil
}
