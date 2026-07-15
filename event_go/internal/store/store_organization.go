package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/emailaddr"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func insertOrganizationTx(tx *sql.Tx, organization *model.Organization, now time.Time) error {
	status := organization.Status
	if status == "" {
		status = model.OrganizationStatusUnclaimed
	}
	result, err := tx.Exec(
		`INSERT INTO organizations (name, slug, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		organization.Name, organization.Slug, status, now.Format(model.TimeFormat), now.Format(model.TimeFormat),
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return model.ErrOrganizationSlugInUse
		}
		return err
	}
	organization.ID, err = result.LastInsertId()
	if err != nil {
		return err
	}
	organization.Status = status
	organization.CreatedAt = now
	organization.UpdatedAt = now
	return nil
}

func insertOrganizerProfileTx(tx *sql.Tx, profile *model.OrganizerProfile, now time.Time) error {
	result, err := tx.Exec(
		`INSERT INTO organizers
		 (organization_id, name, description, contact, logo_url, address, website, tags, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		profile.OrganizationID, profile.Name, profile.Description, profile.Contact,
		profile.LogoURL, profile.Address, profile.Website, profile.Tags,
		now.Format(model.TimeFormat), now.Format(model.TimeFormat),
	)
	if err != nil {
		return err
	}
	profile.ID, err = result.LastInsertId()
	if err != nil {
		return err
	}
	profile.CreatedAt = now
	profile.UpdatedAt = now
	return nil
}

func insertOrganizationMemberTx(tx *sql.Tx, member *model.OrganizationMember, now time.Time) error {
	status := member.Status
	if status == "" {
		status = model.OrganizationMemberStatusActive
	}
	result, err := tx.Exec(
		`INSERT INTO organization_members
		 (organization_id, user_id, role, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		member.OrganizationID, member.UserID, member.Role, status,
		now.Format(model.TimeFormat), now.Format(model.TimeFormat),
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			if member.Role == model.OrganizationRoleOwner && status == model.OrganizationMemberStatusActive {
				return model.ErrOrganizationOwnerExists
			}
			return model.ErrOrganizationMemberExists
		}
		return err
	}
	member.ID, err = result.LastInsertId()
	if err != nil {
		return err
	}
	member.Status = status
	member.CreatedAt = now
	member.UpdatedAt = now
	return nil
}

func (s *Store) CreateOrganizationWithOwner(
	organization *model.Organization,
	profile *model.OrganizerProfile,
	ownerUserID int64,
) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开启组织创建事务失败: %w", err)
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	organization.Status = model.OrganizationStatusActive
	if err := insertOrganizationTx(tx, organization, now); err != nil {
		return fmt.Errorf("创建组织失败: %w", err)
	}
	profile.OrganizationID = organization.ID
	if err := insertOrganizerProfileTx(tx, profile, now); err != nil {
		return fmt.Errorf("创建组织公开资料失败: %w", err)
	}
	member := &model.OrganizationMember{
		OrganizationID: organization.ID,
		UserID:         ownerUserID,
		Role:           model.OrganizationRoleOwner,
	}
	if err := insertOrganizationMemberTx(tx, member, now); err != nil {
		return fmt.Errorf("创建组织所有者失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交组织创建事务失败: %w", err)
	}
	return nil
}

func (s *Store) GetOrganization(id int64) (*model.Organization, error) {
	organization := &model.Organization{}
	var createdAt, updatedAt string
	err := s.db.QueryRow(
		`SELECT id, name, slug, status, created_at, updated_at FROM organizations WHERE id = ?`, id,
	).Scan(
		&organization.ID, &organization.Name, &organization.Slug, &organization.Status,
		&createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询组织失败: %w", err)
	}
	organization.CreatedAt, _ = time.Parse(model.TimeFormat, createdAt)
	organization.UpdatedAt, _ = time.Parse(model.TimeFormat, updatedAt)
	return organization, nil
}

func (s *Store) AddOrganizationMember(member *model.OrganizationMember) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开启成员创建事务失败: %w", err)
	}
	defer tx.Rollback()
	if err := insertOrganizationMemberTx(tx, member, time.Now().UTC()); err != nil {
		return fmt.Errorf("创建组织成员失败: %w", err)
	}
	return tx.Commit()
}

func (s *Store) GetOrganizationMember(organizationID, userID int64) (*model.OrganizationMember, error) {
	member := &model.OrganizationMember{}
	var createdAt, updatedAt string
	err := s.db.QueryRow(
		`SELECT m.id, m.organization_id, o.name, o.slug, o.status, m.user_id, m.role, m.status,
		 m.created_at, m.updated_at
		 FROM organization_members m JOIN organizations o ON o.id = m.organization_id
		 WHERE m.organization_id = ? AND m.user_id = ?`,
		organizationID, userID,
	).Scan(
		&member.ID, &member.OrganizationID, &member.OrganizationName,
		&member.OrganizationSlug, &member.OrganizationStatus, &member.UserID,
		&member.Role, &member.Status, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询组织成员失败: %w", err)
	}
	member.CreatedAt, _ = time.Parse(model.TimeFormat, createdAt)
	member.UpdatedAt, _ = time.Parse(model.TimeFormat, updatedAt)
	return member, nil
}

func (s *Store) ListOrganizationsForUser(userID int64) ([]*model.OrganizationMember, error) {
	rows, err := s.db.Query(
		`SELECT m.id, m.organization_id, o.name, o.slug, o.status, m.user_id, m.role, m.status,
		 m.created_at, m.updated_at
		 FROM organization_members m JOIN organizations o ON o.id = m.organization_id
		 WHERE m.user_id = ? AND m.status = 'active'
		 ORDER BY m.created_at ASC, m.id ASC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询用户组织列表失败: %w", err)
	}
	defer rows.Close()
	result := make([]*model.OrganizationMember, 0)
	for rows.Next() {
		member := &model.OrganizationMember{}
		var createdAt, updatedAt string
		if err := rows.Scan(
			&member.ID, &member.OrganizationID, &member.OrganizationName,
			&member.OrganizationSlug, &member.OrganizationStatus, &member.UserID,
			&member.Role, &member.Status, &createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("读取用户组织列表失败: %w", err)
		}
		member.CreatedAt, _ = time.Parse(model.TimeFormat, createdAt)
		member.UpdatedAt, _ = time.Parse(model.TimeFormat, updatedAt)
		result = append(result, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历用户组织列表失败: %w", err)
	}
	return result, nil
}

func canInviteOrganizationRole(role string) bool {
	switch role {
	case model.OrganizationRoleAdmin, model.OrganizationRoleEditor,
		model.OrganizationRoleChecker, model.OrganizationRoleFinance:
		return true
	default:
		return false
	}
}

func (s *Store) CreateOrganizationInvitation(invitation *model.OrganizationInvitation) error {
	if !canInviteOrganizationRole(invitation.Role) || invitation.ExpiresAt.IsZero() || invitation.TokenHash == "" {
		return model.ErrOrganizationInvitationInvalid
	}
	email, valid := emailaddr.Normalize(invitation.Email)
	if !valid {
		return model.ErrOrganizationInvitationInvalid
	}
	now := time.Now().UTC()
	if !invitation.ExpiresAt.After(now) {
		return model.ErrOrganizationInvitationInvalid
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开启组织邀请事务失败: %w", err)
	}
	defer tx.Rollback()
	inviter := &model.OrganizationMember{}
	if err := tx.QueryRow(
		`SELECT m.role, m.status, o.status
		 FROM organization_members m JOIN organizations o ON o.id = m.organization_id
		 WHERE m.organization_id = ? AND m.user_id = ?`,
		invitation.OrganizationID, invitation.InvitedByUserID,
	).Scan(&inviter.Role, &inviter.Status, &inviter.OrganizationStatus); errors.Is(err, sql.ErrNoRows) {
		return model.ErrOrganizationPermissionDenied
	} else if err != nil {
		return fmt.Errorf("查询邀请人权限失败: %w", err)
	}
	if inviter.OrganizationStatus != model.OrganizationStatusActive ||
		inviter.Status != model.OrganizationMemberStatusActive ||
		(inviter.Role != model.OrganizationRoleOwner && inviter.Role != model.OrganizationRoleAdmin) {
		return model.ErrOrganizationPermissionDenied
	}
	if _, err := tx.Exec(
		`UPDATE organization_invitations
		 SET status = 'expired', updated_at = ?
		 WHERE organization_id = ? AND email = ? AND status = 'pending' AND expires_at <= ?`,
		now.Format(model.TimeFormat), invitation.OrganizationID, email, now.Format(model.TimeFormat),
	); err != nil {
		return fmt.Errorf("清理过期组织邀请失败: %w", err)
	}
	result, err := tx.Exec(
		`INSERT INTO organization_invitations
		 (organization_id, email, role, token_hash, status, expires_at, invited_by_user_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'pending', ?, ?, ?, ?)`,
		invitation.OrganizationID, email, invitation.Role, invitation.TokenHash,
		invitation.ExpiresAt.UTC().Format(model.TimeFormat), invitation.InvitedByUserID,
		now.Format(model.TimeFormat), now.Format(model.TimeFormat),
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return model.ErrOrganizationInvitationExists
		}
		return fmt.Errorf("创建组织邀请失败: %w", err)
	}
	invitation.ID, err = result.LastInsertId()
	if err != nil {
		return err
	}
	invitation.Email = email
	invitation.Status = model.OrganizationInvitationStatusPending
	invitation.CreatedAt = now
	invitation.UpdatedAt = now
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交组织邀请事务失败: %w", err)
	}
	return nil
}

func scanOrganizationInvitation(scanner rowScanner) (*model.OrganizationInvitation, error) {
	invitation := &model.OrganizationInvitation{}
	var expiresAt, createdAt, updatedAt string
	var acceptedAt, revokedAt sql.NullString
	if err := scanner.Scan(
		&invitation.ID, &invitation.OrganizationID, &invitation.Email, &invitation.Role,
		&invitation.TokenHash, &invitation.Status, &expiresAt, &acceptedAt, &revokedAt,
		&invitation.InvitedByUserID, &createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}
	invitation.ExpiresAt, _ = time.Parse(model.TimeFormat, expiresAt)
	invitation.CreatedAt, _ = time.Parse(model.TimeFormat, createdAt)
	invitation.UpdatedAt, _ = time.Parse(model.TimeFormat, updatedAt)
	if acceptedAt.Valid {
		value, err := time.Parse(model.TimeFormat, acceptedAt.String)
		if err != nil {
			return nil, err
		}
		invitation.AcceptedAt = &value
	}
	if revokedAt.Valid {
		value, err := time.Parse(model.TimeFormat, revokedAt.String)
		if err != nil {
			return nil, err
		}
		invitation.RevokedAt = &value
	}
	return invitation, nil
}

func (s *Store) GetOrganizationInvitationByTokenHash(tokenHash string) (*model.OrganizationInvitation, error) {
	invitation, err := scanOrganizationInvitation(s.db.QueryRow(
		`SELECT id, organization_id, email, role, token_hash, status, expires_at,
		 accepted_at, revoked_at, invited_by_user_id, created_at, updated_at
		 FROM organization_invitations WHERE token_hash = ?`, tokenHash,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询组织邀请失败: %w", err)
	}
	return invitation, nil
}

func (s *Store) AcceptOrganizationInvitation(tokenHash string, userID int64, acceptedAt time.Time) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开启接受邀请事务失败: %w", err)
	}
	defer tx.Rollback()
	invitation, err := scanOrganizationInvitation(tx.QueryRow(
		`SELECT id, organization_id, email, role, token_hash, status, expires_at,
		 accepted_at, revoked_at, invited_by_user_id, created_at, updated_at
		 FROM organization_invitations WHERE token_hash = ?`, tokenHash,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return model.ErrOrganizationInvitationInvalid
	}
	if err != nil {
		return fmt.Errorf("查询待接受邀请失败: %w", err)
	}
	acceptedAt = acceptedAt.UTC()
	if invitation.Status != model.OrganizationInvitationStatusPending {
		return model.ErrOrganizationInvitationInvalid
	}
	if !invitation.ExpiresAt.After(acceptedAt) {
		if _, err := tx.Exec(
			`UPDATE organization_invitations
			 SET status = 'expired', updated_at = ?
			 WHERE id = ? AND status = 'pending'`,
			acceptedAt.Format(model.TimeFormat), invitation.ID,
		); err != nil {
			return fmt.Errorf("更新过期组织邀请失败: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("提交过期组织邀请状态失败: %w", err)
		}
		return model.ErrOrganizationInvitationInvalid
	}
	var organizationStatus string
	if err := tx.QueryRow(
		`SELECT status FROM organizations WHERE id = ?`, invitation.OrganizationID,
	).Scan(&organizationStatus); errors.Is(err, sql.ErrNoRows) {
		return model.ErrOrganizationInvitationInvalid
	} else if err != nil {
		return fmt.Errorf("查询邀请组织状态失败: %w", err)
	}
	if organizationStatus != model.OrganizationStatusActive {
		return model.ErrOrganizationInvitationInvalid
	}
	var contact, recoveryEmail string
	var recoveryVerifiedAt sql.NullString
	if err := tx.QueryRow(
		`SELECT contact, recovery_email, recovery_email_verified_at FROM users WHERE id = ?`, userID,
	).Scan(&contact, &recoveryEmail, &recoveryVerifiedAt); err != nil {
		return model.ErrOrganizationInvitationInvalid
	}
	contactEmail, contactIsEmail := emailaddr.Normalize(contact)
	recovery, recoveryIsEmail := emailaddr.Normalize(recoveryEmail)
	if (!contactIsEmail || contactEmail != invitation.Email) &&
		(!recoveryIsEmail || !recoveryVerifiedAt.Valid || recovery != invitation.Email) {
		return model.ErrOrganizationInvitationInvalid
	}
	member := &model.OrganizationMember{
		OrganizationID: invitation.OrganizationID,
		UserID:         userID,
		Role:           invitation.Role,
	}
	if err := insertOrganizationMemberTx(tx, member, acceptedAt); err != nil {
		return err
	}
	result, err := tx.Exec(
		`UPDATE organization_invitations
		 SET status = 'accepted', accepted_at = ?, updated_at = ?
		 WHERE id = ? AND status = 'pending'`,
		acceptedAt.Format(model.TimeFormat), acceptedAt.Format(model.TimeFormat), invitation.ID,
	)
	if err != nil {
		return fmt.Errorf("接受组织邀请失败: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil || updated != 1 {
		return model.ErrOrganizationInvitationInvalid
	}
	return tx.Commit()
}
