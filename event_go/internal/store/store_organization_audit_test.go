package store

import (
	"strings"
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func TestOrganizationAuditIsScopedAndImmutable(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	owner := &model.User{Name: "审计所有者", Contact: "audit-owner@example.com", PasswordHash: "hash"}
	if err := s.CreateUser(owner); err != nil {
		t.Fatal(err)
	}
	organization := &model.Organization{Name: "审计组织", Slug: "audit-org"}
	profile := &model.OrganizerProfile{Name: "审计资料"}
	if err := s.CreateOrganizationWithOwner(organization, profile, owner.ID); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2031, 2, 3, 4, 5, 6, 0, time.UTC)
	entry := &model.OrganizationAuditLog{
		OrganizationID: organization.ID,
		ActorType:      model.AuditActorOrganizationMember, ActorID: &owner.ID,
		Action: "updateOrganizationEvent", ResourceType: "event", ResourceID: "42",
		RequestID: "request-audit-001", Outcome: model.AuditOutcomeSuccess,
		HTTPStatus: 200, CreatedAt: now,
	}
	if err := s.AppendOrganizationAudit(entry); err != nil {
		t.Fatal(err)
	}
	entries, total, err := s.ListOrganizationAudits(organization.ID, 0, 20)
	if err != nil || total != 1 || len(entries) != 1 {
		t.Fatalf("audit list mismatch: entries=%+v total=%d err=%v", entries, total, err)
	}
	got := entries[0]
	if got.RequestID != entry.RequestID || got.ResourceID != "42" || got.ActorID == nil || *got.ActorID != owner.ID || !got.CreatedAt.Equal(now) {
		t.Fatalf("audit entry mismatch: %+v", got)
	}
	if _, err := s.db.Exec(`UPDATE organization_audit_logs SET outcome = 'failure' WHERE id = ?`, entry.ID); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("audit update was not rejected: %v", err)
	}
	if _, err := s.db.Exec(`DELETE FROM organization_audit_logs WHERE id = ?`, entry.ID); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("audit delete was not rejected: %v", err)
	}
}

func TestOrganizationAuditRejectsInvalidAndCrossTenantReads(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.AppendOrganizationAudit(&model.OrganizationAuditLog{}); err == nil {
		t.Fatal("invalid audit entry was accepted")
	}
	entries, total, err := s.ListOrganizationAudits(999, 0, 20)
	if err != nil || total != 0 || len(entries) != 0 {
		t.Fatalf("missing tenant audit list mismatch: entries=%+v total=%d err=%v", entries, total, err)
	}
}
