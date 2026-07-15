package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/clock"
	"github.com/qw2261/soulmarker/event_go/internal/emailaddr"
	"github.com/qw2261/soulmarker/event_go/internal/identifier"
	"github.com/qw2261/soulmarker/event_go/internal/model"
	"github.com/qw2261/soulmarker/event_go/internal/notification"
)

var (
	ErrOrganizationInputInvalid = errors.New("组织信息格式无效")
	ErrOrganizationRoleInvalid  = errors.New("组织角色无效")
)

var organizationSlugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,46}[a-z0-9])?$`)

type OrganizationSelfServiceRepository interface {
	CreateOrganizationWithOwner(organization *model.Organization, profile *model.OrganizerProfile, ownerUserID int64) error
	GetOrganization(id int64) (*model.Organization, error)
	GetOrganizerProfileForOrganization(organizationID int64) (*model.OrganizerProfile, error)
	ListOrganizationMembers(organizationID int64) ([]*model.OrganizationMember, error)
	UpdateOrganizationMemberRole(organizationID, actorUserID, memberID int64, role string, updatedAt time.Time) error
	RevokeOrganizationMember(organizationID, actorUserID, memberID int64, revokedAt time.Time) error
	CreateOrganizationInvitation(invitation *model.OrganizationInvitation) error
	ListOrganizationInvitations(organizationID int64, now time.Time) ([]*model.OrganizationInvitation, error)
	RevokeOrganizationInvitation(organizationID, actorUserID, invitationID int64, revokedAt time.Time) error
	AcceptOrganizationInvitation(tokenHash string, userID int64, acceptedAt time.Time) error
}

type OrganizationSelfService struct {
	repository OrganizationSelfServiceRepository
	clock      clock.Clock
	tokens     identifier.ResetTokenGenerator
	sender     notification.OrganizationInvitationSender
	baseURL    string
	inviteTTL  time.Duration
}

func NewOrganizationSelfService(
	repository OrganizationSelfServiceRepository,
	businessClock clock.Clock,
	tokens identifier.ResetTokenGenerator,
	sender notification.OrganizationInvitationSender,
	baseURL string,
	inviteTTL time.Duration,
) *OrganizationSelfService {
	return &OrganizationSelfService{
		repository: repository, clock: businessClock, tokens: tokens, sender: sender,
		baseURL: strings.TrimRight(baseURL, "/"), inviteTTL: inviteTTL,
	}
}

func normalizeOrganizationInput(organization *model.Organization, profile *model.OrganizerProfile) error {
	organization.Name = strings.TrimSpace(organization.Name)
	organization.Slug = strings.ToLower(strings.TrimSpace(organization.Slug))
	profile.Name = strings.TrimSpace(profile.Name)
	profile.Description = strings.TrimSpace(profile.Description)
	profile.Contact = strings.TrimSpace(profile.Contact)
	profile.LogoURL = strings.TrimSpace(profile.LogoURL)
	profile.Address = strings.TrimSpace(profile.Address)
	profile.Website = strings.TrimSpace(profile.Website)
	profile.Tags = strings.TrimSpace(profile.Tags)
	if len([]rune(organization.Name)) < 2 || len([]rune(organization.Name)) > 100 ||
		!organizationSlugPattern.MatchString(organization.Slug) ||
		len([]rune(profile.Name)) < 2 || len([]rune(profile.Name)) > 100 {
		return ErrOrganizationInputInvalid
	}
	return nil
}

func (s *OrganizationSelfService) Create(
	ownerUserID int64,
	organization *model.Organization,
	profile *model.OrganizerProfile,
) error {
	if ownerUserID <= 0 || normalizeOrganizationInput(organization, profile) != nil {
		return ErrOrganizationInputInvalid
	}
	return s.repository.CreateOrganizationWithOwner(organization, profile, ownerUserID)
}

func (s *OrganizationSelfService) Get(organizationID int64) (*model.Organization, *model.OrganizerProfile, error) {
	organization, err := s.repository.GetOrganization(organizationID)
	if err != nil || organization == nil {
		return organization, nil, err
	}
	profile, err := s.repository.GetOrganizerProfileForOrganization(organizationID)
	return organization, profile, err
}

func (s *OrganizationSelfService) ListMembers(organizationID int64) ([]*model.OrganizationMember, error) {
	return s.repository.ListOrganizationMembers(organizationID)
}

func (s *OrganizationSelfService) UpdateMemberRole(organizationID, actorUserID, memberID int64, role string) error {
	if !canSelfServiceRole(role) {
		return ErrOrganizationRoleInvalid
	}
	return s.repository.UpdateOrganizationMemberRole(organizationID, actorUserID, memberID, role, s.clock.Now())
}

func (s *OrganizationSelfService) RevokeMember(organizationID, actorUserID, memberID int64) error {
	return s.repository.RevokeOrganizationMember(organizationID, actorUserID, memberID, s.clock.Now())
}

func canSelfServiceRole(role string) bool {
	switch role {
	case model.OrganizationRoleAdmin, model.OrganizationRoleEditor,
		model.OrganizationRoleChecker, model.OrganizationRoleFinance:
		return true
	default:
		return false
	}
}

func (s *OrganizationSelfService) Invite(
	ctx context.Context,
	organizationID, actorUserID int64,
	email, role string,
) (*model.OrganizationInvitation, error) {
	normalizedEmail, valid := emailaddr.Normalize(email)
	if !valid {
		return nil, ErrOrganizationInputInvalid
	}
	if !canSelfServiceRole(role) {
		return nil, ErrOrganizationRoleInvalid
	}
	organization, err := s.repository.GetOrganization(organizationID)
	if err != nil {
		return nil, err
	}
	if organization == nil || organization.Status != model.OrganizationStatusActive {
		return nil, model.ErrOrganizationPermissionDenied
	}
	rawToken, err := s.tokens.NewResetToken()
	if err != nil {
		return nil, fmt.Errorf("generate organization invitation token: %w", err)
	}
	now := s.clock.Now().UTC()
	invitationURL, err := url.Parse(s.baseURL + "/organization-invitations/accept")
	if err != nil {
		return nil, fmt.Errorf("build organization invitation URL: %w", err)
	}
	query := invitationURL.Query()
	query.Set("token", rawToken)
	invitationURL.RawQuery = query.Encode()
	invitation := &model.OrganizationInvitation{
		OrganizationID: organizationID, Email: normalizedEmail, Role: role,
		TokenHash: resetTokenHash(rawToken), ExpiresAt: now.Add(s.inviteTTL), InvitedByUserID: actorUserID,
	}
	if err := s.repository.CreateOrganizationInvitation(invitation); err != nil {
		return nil, err
	}
	if err := s.sender.SendOrganizationInvitation(
		ctx, normalizedEmail, organization.Name, role, invitationURL.String(), invitation.ExpiresAt,
	); err != nil {
		if revokeErr := s.repository.RevokeOrganizationInvitation(organizationID, actorUserID, invitation.ID, now); revokeErr != nil {
			return nil, fmt.Errorf("deliver organization invitation: %w; revoke invitation: %v", err, revokeErr)
		}
		return nil, fmt.Errorf("deliver organization invitation: %w", err)
	}
	return invitation, nil
}

func (s *OrganizationSelfService) ListInvitations(organizationID int64) ([]*model.OrganizationInvitation, error) {
	return s.repository.ListOrganizationInvitations(organizationID, s.clock.Now())
}

func (s *OrganizationSelfService) RevokeInvitation(organizationID, actorUserID, invitationID int64) error {
	return s.repository.RevokeOrganizationInvitation(organizationID, actorUserID, invitationID, s.clock.Now())
}

func (s *OrganizationSelfService) Accept(rawToken string, userID int64) error {
	if strings.TrimSpace(rawToken) == "" || userID <= 0 {
		return model.ErrOrganizationInvitationInvalid
	}
	return s.repository.AcceptOrganizationInvitation(resetTokenHash(rawToken), userID, s.clock.Now())
}
