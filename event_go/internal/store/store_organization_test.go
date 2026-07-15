package store

import (
	"errors"
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func TestOrganizationOwnerMembershipAndProfileAreAtomic(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	owner := &model.User{Name: "组织所有者", Contact: "owner@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(owner); err != nil {
		t.Fatal(err)
	}
	organization := &model.Organization{Name: "Alpha 组织", Slug: "alpha"}
	profile := &model.OrganizerProfile{Name: "Alpha 门店", Contact: "public@example.com"}
	if err := s.CreateOrganizationWithOwner(organization, profile, owner.ID); err != nil {
		t.Fatal(err)
	}
	if organization.ID == 0 || organization.Status != model.OrganizationStatusActive || profile.OrganizationID != organization.ID {
		t.Fatalf("unexpected organization/profile: organization=%+v profile=%+v", organization, profile)
	}
	member, err := s.GetOrganizationMember(organization.ID, owner.ID)
	if err != nil || member == nil || member.Role != model.OrganizationRoleOwner || member.Status != model.OrganizationMemberStatusActive {
		t.Fatalf("owner membership missing: member=%+v err=%v", member, err)
	}
	organizations, err := s.ListOrganizationsForUser(owner.ID)
	if err != nil || len(organizations) != 1 || organizations[0].OrganizationSlug != "alpha" {
		t.Fatalf("owner organization list mismatch: organizations=%+v err=%v", organizations, err)
	}
	secondOwner := &model.User{Name: "第二所有者", Contact: "second-owner@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(secondOwner); err != nil {
		t.Fatal(err)
	}
	if err := s.AddOrganizationMember(&model.OrganizationMember{
		OrganizationID: organization.ID, UserID: secondOwner.ID, Role: "superadmin",
	}); err == nil {
		t.Fatal("organization accepted an undefined role")
	}
	if err := s.AddOrganizationMember(&model.OrganizationMember{
		OrganizationID: organization.ID, UserID: secondOwner.ID, Role: model.OrganizationRoleOwner,
	}); !errors.Is(err, model.ErrOrganizationOwnerExists) {
		t.Fatalf("organization accepted a second active owner: %v", err)
	}

	missingOwnerOrganization := &model.Organization{Name: "回滚组织", Slug: "rollback-org"}
	missingOwnerProfile := &model.OrganizerProfile{Name: "不应存在的资料"}
	if err := s.CreateOrganizationWithOwner(missingOwnerOrganization, missingOwnerProfile, 99999); err == nil {
		t.Fatal("organization without a real owner was created")
	}
	if got, err := s.GetOrganization(missingOwnerOrganization.ID); err != nil || got != nil {
		t.Fatalf("failed owner transaction left organization behind: organization=%+v err=%v", got, err)
	}
}

func TestOrganizationInvitationEnforcesInviterRoleEmailAndSingleUse(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	owner := &model.User{Name: "所有者", Contact: "owner-invite@example.com", PasswordHash: "hash"}
	invitee := &model.User{Name: "受邀者", Contact: "Invitee@Example.COM", PasswordHash: "hash"}
	other := &model.User{Name: "错误账户", Contact: "other-invite@example.com", PasswordHash: "hash"}
	admin := &model.User{Name: "管理员", Contact: "admin-invite@example.com", PasswordHash: "hash"}
	for _, user := range []*model.User{owner, invitee, other, admin} {
		if err := s.CreateUser(user); err != nil {
			t.Fatal(err)
		}
	}
	organization := &model.Organization{Name: "邀请组织", Slug: "invite-org"}
	profile := &model.OrganizerProfile{Name: "邀请门店"}
	if err := s.CreateOrganizationWithOwner(organization, profile, owner.ID); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	invitation := &model.OrganizationInvitation{
		OrganizationID:  organization.ID,
		Email:           "INVITEE@example.com",
		Role:            model.OrganizationRoleEditor,
		TokenHash:       "invite-token-hash-1",
		ExpiresAt:       now.Add(time.Hour),
		InvitedByUserID: owner.ID,
	}
	if err := s.CreateOrganizationInvitation(invitation); err != nil {
		t.Fatal(err)
	}
	if invitation.Email != "invitee@example.com" || invitation.Status != model.OrganizationInvitationStatusPending {
		t.Fatalf("invitation not normalized: %+v", invitation)
	}
	duplicate := *invitation
	duplicate.ID = 0
	duplicate.TokenHash = "invite-token-hash-duplicate"
	if err := s.CreateOrganizationInvitation(&duplicate); !errors.Is(err, model.ErrOrganizationInvitationExists) {
		t.Fatalf("duplicate pending invitation allowed: %v", err)
	}
	if err := s.AcceptOrganizationInvitation(invitation.TokenHash, other.ID, now.Add(time.Minute)); !errors.Is(err, model.ErrOrganizationInvitationInvalid) {
		t.Fatalf("wrong email accepted invitation: %v", err)
	}
	if err := s.AcceptOrganizationInvitation(invitation.TokenHash, invitee.ID, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	member, err := s.GetOrganizationMember(organization.ID, invitee.ID)
	if err != nil || member == nil || member.Role != model.OrganizationRoleEditor {
		t.Fatalf("accepted member missing: member=%+v err=%v", member, err)
	}
	if err := s.AcceptOrganizationInvitation(invitation.TokenHash, invitee.ID, now.Add(3*time.Minute)); !errors.Is(err, model.ErrOrganizationInvitationInvalid) {
		t.Fatalf("invitation was reusable: %v", err)
	}
	stored, err := s.GetOrganizationInvitationByTokenHash(invitation.TokenHash)
	if err != nil || stored == nil || stored.Status != model.OrganizationInvitationStatusAccepted || stored.AcceptedAt == nil {
		t.Fatalf("accepted invitation state mismatch: invitation=%+v err=%v", stored, err)
	}

	invalidOwnerInvite := &model.OrganizationInvitation{
		OrganizationID: organization.ID, Email: "new-owner@example.com",
		Role: model.OrganizationRoleOwner, TokenHash: "owner-token",
		ExpiresAt: now.Add(time.Hour), InvitedByUserID: owner.ID,
	}
	if err := s.CreateOrganizationInvitation(invalidOwnerInvite); !errors.Is(err, model.ErrOrganizationInvitationInvalid) {
		t.Fatalf("owner invitation must require explicit transfer flow: %v", err)
	}
	unauthorizedInvite := &model.OrganizationInvitation{
		OrganizationID: organization.ID, Email: "finance@example.com",
		Role: model.OrganizationRoleFinance, TokenHash: "unauthorized-token",
		ExpiresAt: now.Add(time.Hour), InvitedByUserID: invitee.ID,
	}
	if err := s.CreateOrganizationInvitation(unauthorizedInvite); !errors.Is(err, model.ErrOrganizationPermissionDenied) {
		t.Fatalf("editor created invitation: %v", err)
	}
	if err := s.AddOrganizationMember(&model.OrganizationMember{
		OrganizationID: organization.ID, UserID: admin.ID, Role: model.OrganizationRoleAdmin,
	}); err != nil {
		t.Fatal(err)
	}
	adminInvitation := &model.OrganizationInvitation{
		OrganizationID: organization.ID, Email: "finance@example.com",
		Role: model.OrganizationRoleFinance, TokenHash: "admin-created-token",
		ExpiresAt: now.Add(time.Hour), InvitedByUserID: admin.ID,
	}
	if err := s.CreateOrganizationInvitation(adminInvitation); err != nil {
		t.Fatalf("active admin could not create invitation: %v", err)
	}
}

func TestOrganizationInvitationSupportsVerifiedRecoveryEmailAndExpiresPendingInvites(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	owner := &model.User{Name: "所有者", Contact: "recovery-owner@example.com", PasswordHash: "hash"}
	invitee := &model.User{Name: "恢复邮箱受邀者", Contact: "13800138000", PasswordHash: "hash"}
	for _, user := range []*model.User{owner, invitee} {
		if err := s.CreateUser(user); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.db.Exec(
		`UPDATE users SET recovery_email = ? WHERE id = ?`,
		"billing@example.com", invitee.ID,
	); err != nil {
		t.Fatal(err)
	}
	organization := &model.Organization{Name: "恢复邮箱组织", Slug: "recovery-invite"}
	profile := &model.OrganizerProfile{Name: "恢复邮箱门店"}
	if err := s.CreateOrganizationWithOwner(organization, profile, owner.ID); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	expired := &model.OrganizationInvitation{
		OrganizationID: organization.ID, Email: "billing@example.com",
		Role: model.OrganizationRoleFinance, TokenHash: "expired-recovery-token",
		ExpiresAt: now.Add(-time.Minute), InvitedByUserID: owner.ID,
	}
	if _, err := s.db.Exec(
		`INSERT INTO organization_invitations
		 (organization_id, email, role, token_hash, status, expires_at, invited_by_user_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'pending', ?, ?, ?, ?)`,
		expired.OrganizationID, expired.Email, expired.Role, expired.TokenHash,
		expired.ExpiresAt.Format(model.TimeFormat), expired.InvitedByUserID,
		now.Add(-time.Hour).Format(model.TimeFormat), now.Add(-time.Hour).Format(model.TimeFormat),
	); err != nil {
		t.Fatal(err)
	}
	replacement := &model.OrganizationInvitation{
		OrganizationID: organization.ID, Email: "BILLING@example.com",
		Role: model.OrganizationRoleFinance, TokenHash: "replacement-recovery-token",
		ExpiresAt: now.Add(time.Hour), InvitedByUserID: owner.ID,
	}
	if err := s.CreateOrganizationInvitation(replacement); err != nil {
		t.Fatalf("expired pending invitation blocked replacement: %v", err)
	}
	storedExpired, err := s.GetOrganizationInvitationByTokenHash(expired.TokenHash)
	if err != nil || storedExpired == nil || storedExpired.Status != model.OrganizationInvitationStatusExpired {
		t.Fatalf("expired invitation state mismatch: invitation=%+v err=%v", storedExpired, err)
	}
	if err := s.AcceptOrganizationInvitation(replacement.TokenHash, invitee.ID, now.Add(time.Minute)); !errors.Is(err, model.ErrOrganizationInvitationInvalid) {
		t.Fatalf("unverified recovery email accepted invitation: %v", err)
	}
	verifiedAt := now.Format(model.TimeFormat)
	if _, err := s.db.Exec(
		`UPDATE users SET recovery_email_verified_at = ? WHERE id = ?`, verifiedAt, invitee.ID,
	); err != nil {
		t.Fatal(err)
	}
	if err := s.AcceptOrganizationInvitation(replacement.TokenHash, invitee.ID, now.Add(time.Minute)); err != nil {
		t.Fatalf("verified recovery email did not accept invitation: %v", err)
	}
}

func TestOrganizationMemberManagementProtectsOwnerAndSupportsReinvite(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	owner := &model.User{Name: "成员管理所有者", Contact: "manage-owner@example.com", PasswordHash: "hash"}
	admin := &model.User{Name: "成员管理员", Contact: "manage-admin@example.com", PasswordHash: "hash"}
	editor := &model.User{Name: "成员编辑", Contact: "manage-editor@example.com", PasswordHash: "hash"}
	for _, user := range []*model.User{owner, admin, editor} {
		if err := s.CreateUser(user); err != nil {
			t.Fatal(err)
		}
	}
	organization := &model.Organization{Name: "成员管理组织", Slug: "member-management"}
	profile := &model.OrganizerProfile{Name: "成员管理门店"}
	if err := s.CreateOrganizationWithOwner(organization, profile, owner.ID); err != nil {
		t.Fatal(err)
	}
	adminMember := &model.OrganizationMember{OrganizationID: organization.ID, UserID: admin.ID, Role: model.OrganizationRoleAdmin}
	editorMember := &model.OrganizationMember{OrganizationID: organization.ID, UserID: editor.ID, Role: model.OrganizationRoleEditor}
	if err := s.AddOrganizationMember(adminMember); err != nil {
		t.Fatal(err)
	}
	if err := s.AddOrganizationMember(editorMember); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := s.UpdateOrganizationMemberRole(organization.ID, admin.ID, editorMember.ID, model.OrganizationRoleChecker, now); err != nil {
		t.Fatalf("admin could not manage editor: %v", err)
	}
	if err := s.UpdateOrganizationMemberRole(organization.ID, admin.ID, editorMember.ID, model.OrganizationRoleAdmin, now); !errors.Is(err, model.ErrOrganizationMemberChangeDenied) {
		t.Fatalf("admin granted admin role: %v", err)
	}
	ownerMember, err := s.GetOrganizationMember(organization.ID, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RevokeOrganizationMember(organization.ID, admin.ID, ownerMember.ID, now); !errors.Is(err, model.ErrOrganizationMemberChangeDenied) {
		t.Fatalf("admin revoked owner: %v", err)
	}
	if err := s.RevokeOrganizationMember(organization.ID, admin.ID, adminMember.ID, now); !errors.Is(err, model.ErrOrganizationMemberChangeDenied) {
		t.Fatalf("admin revoked self: %v", err)
	}
	if err := s.RevokeOrganizationMember(organization.ID, owner.ID, adminMember.ID, now); err != nil {
		t.Fatalf("owner could not revoke admin: %v", err)
	}
	reinvite := &model.OrganizationInvitation{
		OrganizationID: organization.ID, Email: admin.Contact, Role: model.OrganizationRoleFinance,
		TokenHash: "reinvite-revoked-admin", ExpiresAt: now.Add(time.Hour), InvitedByUserID: owner.ID,
	}
	if err := s.CreateOrganizationInvitation(reinvite); err != nil {
		t.Fatal(err)
	}
	if err := s.AcceptOrganizationInvitation(reinvite.TokenHash, admin.ID, now.Add(time.Minute)); err != nil {
		t.Fatalf("revoked member could not accept reinvite: %v", err)
	}
	restored, err := s.GetOrganizationMember(organization.ID, admin.ID)
	if err != nil || restored == nil || restored.Status != model.OrganizationMemberStatusActive || restored.Role != model.OrganizationRoleFinance {
		t.Fatalf("reinvited member mismatch: member=%+v err=%v", restored, err)
	}
	members, err := s.ListOrganizationMembers(organization.ID)
	if err != nil || len(members) != 3 || members[0].UserName == "" || members[0].UserContact == "" {
		t.Fatalf("member list missing user projection: members=%+v err=%v", members, err)
	}
}

func TestOrganizationInvitationRejectsSuspendedOrganization(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	owner := &model.User{Name: "暂停组织所有者", Contact: "suspended-owner@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(owner); err != nil {
		t.Fatal(err)
	}
	organization := &model.Organization{Name: "暂停组织", Slug: "suspended-org"}
	profile := &model.OrganizerProfile{Name: "暂停门店"}
	if err := s.CreateOrganizationWithOwner(organization, profile, owner.ID); err != nil {
		t.Fatal(err)
	}
	invitation := &model.OrganizationInvitation{
		OrganizationID: organization.ID, Email: owner.Contact,
		Role: model.OrganizationRoleEditor, TokenHash: "suspended-token",
		ExpiresAt: time.Now().UTC().Add(time.Hour), InvitedByUserID: owner.ID,
	}
	if err := s.CreateOrganizationInvitation(invitation); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE organizations SET status = 'suspended' WHERE id = ?`, organization.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.AcceptOrganizationInvitation(invitation.TokenHash, owner.ID, time.Now().UTC()); !errors.Is(err, model.ErrOrganizationInvitationInvalid) {
		t.Fatalf("suspended organization accepted invitation: %v", err)
	}
	secondInvitation := &model.OrganizationInvitation{
		OrganizationID: organization.ID, Email: "member@example.com",
		Role: model.OrganizationRoleEditor, TokenHash: "suspended-token-2",
		ExpiresAt: time.Now().UTC().Add(time.Hour), InvitedByUserID: owner.ID,
	}
	if err := s.CreateOrganizationInvitation(secondInvitation); !errors.Is(err, model.ErrOrganizationPermissionDenied) {
		t.Fatalf("suspended organization created invitation: %v", err)
	}
}

func TestLegacyOrganizerCreatesUnclaimedTenantAndDeletionSuspendsIt(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	profile := &model.Organizer{Name: "兼容门店"}
	if err := s.CreateOrganizer(profile); err != nil {
		t.Fatal(err)
	}
	organization, err := s.GetOrganization(profile.OrganizationID)
	if err != nil || organization == nil || organization.Status != model.OrganizationStatusUnclaimed {
		t.Fatalf("legacy create did not produce unclaimed tenant: organization=%+v err=%v", organization, err)
	}
	if err := s.DeleteOrganizer(profile.ID); err != nil {
		t.Fatal(err)
	}
	organization, err = s.GetOrganization(profile.OrganizationID)
	if err != nil || organization == nil || organization.Status != model.OrganizationStatusSuspended {
		t.Fatalf("deleted profile tenant was not suspended: organization=%+v err=%v", organization, err)
	}
}

func TestPreV12OrganizerInsertRemainsWriteCompatible(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Now().UTC().Format(model.TimeFormat)
	result, err := s.db.Exec(
		`INSERT INTO organizers
		 (name, description, contact, logo_url, address, website, tags, created_at, updated_at)
		 VALUES ('旧应用门店', '', '', '', '', '', '', ?, ?)`, now, now,
	)
	if err != nil {
		t.Fatalf("pre-v12 organizer insert failed: %v", err)
	}
	profileID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	profile, err := s.GetOrganizer(profileID)
	if err != nil || profile == nil || profile.OrganizationID == 0 {
		t.Fatalf("compatibility trigger did not assign tenant: profile=%+v err=%v", profile, err)
	}
	organization, err := s.GetOrganization(profile.OrganizationID)
	if err != nil || organization == nil || organization.Status != model.OrganizationStatusUnclaimed || organization.Name != profile.Name {
		t.Fatalf("compatibility tenant mismatch: organization=%+v err=%v", organization, err)
	}
	if _, err := s.db.Exec(`DELETE FROM organizers WHERE id = ?`, profile.ID); err != nil {
		t.Fatalf("pre-v12 organizer delete failed: %v", err)
	}
	organization, err = s.GetOrganization(profile.OrganizationID)
	if err != nil || organization == nil || organization.Status != model.OrganizationStatusSuspended {
		t.Fatalf("compatibility delete did not suspend tenant: organization=%+v err=%v", organization, err)
	}
}
