package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
	"github.com/qw2261/soulmarker/event_go/internal/store"
)

type recordingOrganizationInvitationSender struct {
	email, organization, role, invitationURL string
	expiresAt                                time.Time
	err                                      error
}

func (s *recordingOrganizationInvitationSender) SendOrganizationInvitation(
	_ context.Context,
	email, organizationName, role, invitationURL string,
	expiresAt time.Time,
) error {
	s.email, s.organization, s.role, s.invitationURL, s.expiresAt =
		email, organizationName, role, invitationURL, expiresAt
	return s.err
}

func TestOrganizationSelfServiceCreatesInvitesAndAcceptsWithoutStoringRawToken(t *testing.T) {
	repository, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	owner := &model.User{Name: "自助所有者", Contact: "self-owner@example.com", PasswordHash: "hash"}
	invitee := &model.User{Name: "自助成员", Contact: "self-member@example.com", PasswordHash: "hash"}
	for _, user := range []*model.User{owner, invitee} {
		if err := repository.CreateUser(user); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	tokens := &fixedResetTokenGenerator{token: "raw-invitation-token"}
	sender := &recordingOrganizationInvitationSender{}
	selfService := NewOrganizationSelfService(
		repository, fixedClock{now: now}, tokens, sender,
		"https://events.example.com/", 72*time.Hour,
	)
	organization := &model.Organization{Name: " 自助组织 ", Slug: "SELF-SERVICE"}
	profile := &model.OrganizerProfile{Name: " 自助门店 ", Contact: "public@example.com"}
	if err := selfService.Create(owner.ID, organization, profile); err != nil {
		t.Fatal(err)
	}
	if organization.Status != model.OrganizationStatusActive || organization.Slug != "self-service" || profile.OrganizationID != organization.ID {
		t.Fatalf("self-service organization mismatch: organization=%+v profile=%+v", organization, profile)
	}
	invitation, err := selfService.Invite(
		context.Background(), organization.ID, owner.ID,
		"SELF-MEMBER@example.com", model.OrganizationRoleEditor,
	)
	if err != nil {
		t.Fatal(err)
	}
	if tokens.calls != 1 || sender.email != "self-member@example.com" || sender.organization != organization.Name ||
		!strings.Contains(sender.invitationURL, "token=raw-invitation-token") || !sender.expiresAt.Equal(now.Add(72*time.Hour)) {
		t.Fatalf("invitation delivery mismatch: sender=%+v calls=%d", sender, tokens.calls)
	}
	if invitation.TokenHash == "raw-invitation-token" || invitation.TokenHash != resetTokenHash("raw-invitation-token") {
		t.Fatalf("raw invitation token was stored: %+v", invitation)
	}
	if _, err := selfService.Accept("raw-invitation-token", invitee.ID); err != nil {
		t.Fatal(err)
	}
	members, err := selfService.ListMembers(organization.ID)
	if err != nil || len(members) != 2 || members[1].UserID != invitee.ID || members[1].Role != model.OrganizationRoleEditor {
		t.Fatalf("accepted member mismatch: members=%+v err=%v", members, err)
	}
	if _, err := selfService.Accept("raw-invitation-token", invitee.ID); !errors.Is(err, model.ErrOrganizationInvitationInvalid) {
		t.Fatalf("invitation was reusable: %v", err)
	}
}

func TestOrganizationSelfServiceRevokesInvitationWhenDeliveryFails(t *testing.T) {
	repository, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	owner := &model.User{Name: "投递失败所有者", Contact: "delivery-owner@example.com", PasswordHash: "hash"}
	if err := repository.CreateUser(owner); err != nil {
		t.Fatal(err)
	}
	organization := &model.Organization{Name: "投递失败组织", Slug: "delivery-failure"}
	profile := &model.OrganizerProfile{Name: "投递失败门店"}
	if err := repository.CreateOrganizationWithOwner(organization, profile, owner.ID); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	selfService := NewOrganizationSelfService(
		repository, fixedClock{now: now}, &fixedResetTokenGenerator{token: "failed-token"},
		&recordingOrganizationInvitationSender{err: errors.New("smtp unavailable")},
		"https://events.example.com", time.Hour,
	)
	if _, err := selfService.Invite(
		context.Background(), organization.ID, owner.ID,
		"delivery-member@example.com", model.OrganizationRoleFinance,
	); err == nil {
		t.Fatal("delivery failure was ignored")
	}
	invitations, err := selfService.ListInvitations(organization.ID)
	if err != nil || len(invitations) != 1 || invitations[0].Status != model.OrganizationInvitationStatusRevoked {
		t.Fatalf("failed delivery invitation remained pending: invitations=%+v err=%v", invitations, err)
	}
}
