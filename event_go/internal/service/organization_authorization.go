package service

import (
	"fmt"

	"github.com/qw2261/soulmarker/event_go/internal/authorization"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

type OrganizationAuthorizationRepository interface {
	GetOrganizationMember(organizationID, userID int64) (*model.OrganizationMember, error)
	ListOrganizationsForUser(userID int64) ([]*model.OrganizationMember, error)
}

type OrganizationAuthorizationService struct {
	repository OrganizationAuthorizationRepository
}

func NewOrganizationAuthorizationService(repository OrganizationAuthorizationRepository) *OrganizationAuthorizationService {
	return &OrganizationAuthorizationService{repository: repository}
}

func organizationContext(member *model.OrganizationMember) authorization.OrganizationContext {
	capabilities := []authorization.Capability{}
	if member.Status == model.OrganizationMemberStatusActive && member.OrganizationStatus == model.OrganizationStatusActive {
		capabilities = authorization.CapabilitiesForRole(member.Role)
	}
	return authorization.OrganizationContext{
		OrganizationID:     member.OrganizationID,
		OrganizationName:   member.OrganizationName,
		OrganizationSlug:   member.OrganizationSlug,
		OrganizationStatus: member.OrganizationStatus,
		MembershipStatus:   member.Status,
		UserID:             member.UserID,
		Role:               member.Role,
		Capabilities:       capabilities,
	}
}

func (s *OrganizationAuthorizationService) ListForUser(userID int64) ([]authorization.OrganizationContext, error) {
	memberships, err := s.repository.ListOrganizationsForUser(userID)
	if err != nil {
		return nil, fmt.Errorf("list organization authorization contexts: %w", err)
	}
	result := make([]authorization.OrganizationContext, 0, len(memberships))
	for _, membership := range memberships {
		result = append(result, organizationContext(membership))
	}
	return result, nil
}

func (s *OrganizationAuthorizationService) Authorize(
	userID, organizationID int64,
	capability authorization.Capability,
) (*authorization.OrganizationContext, error) {
	if userID <= 0 || organizationID <= 0 {
		return nil, authorization.ErrOrganizationAccessDenied
	}
	membership, err := s.repository.GetOrganizationMember(organizationID, userID)
	if err != nil {
		return nil, fmt.Errorf("load organization membership: %w", err)
	}
	if membership == nil || membership.Status != model.OrganizationMemberStatusActive ||
		membership.OrganizationStatus != model.OrganizationStatusActive ||
		!authorization.Allows(membership.Role, capability) {
		return nil, authorization.ErrOrganizationAccessDenied
	}
	value := organizationContext(membership)
	return &value, nil
}
