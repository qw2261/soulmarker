package service

import (
	"errors"
	"testing"

	"github.com/qw2261/soulmarker/event_go/internal/authorization"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

type organizationAuthorizationRepositoryStub struct {
	memberships []*model.OrganizationMember
	err         error
}

func (r *organizationAuthorizationRepositoryStub) GetOrganizationMember(organizationID, userID int64) (*model.OrganizationMember, error) {
	if r.err != nil {
		return nil, r.err
	}
	for _, membership := range r.memberships {
		if membership.OrganizationID == organizationID && membership.UserID == userID {
			copy := *membership
			return &copy, nil
		}
	}
	return nil, nil
}

func (r *organizationAuthorizationRepositoryStub) ListOrganizationsForUser(userID int64) ([]*model.OrganizationMember, error) {
	if r.err != nil {
		return nil, r.err
	}
	result := make([]*model.OrganizationMember, 0)
	for _, membership := range r.memberships {
		if membership.UserID == userID && membership.Status == model.OrganizationMemberStatusActive {
			copy := *membership
			result = append(result, &copy)
		}
	}
	return result, nil
}

func TestOrganizationAuthorizationServiceRejectsCrossTenantAndInactiveAccess(t *testing.T) {
	repository := &organizationAuthorizationRepositoryStub{memberships: []*model.OrganizationMember{
		{
			OrganizationID: 1, OrganizationName: "Alpha", OrganizationStatus: model.OrganizationStatusActive,
			UserID: 10, Role: model.OrganizationRoleEditor, Status: model.OrganizationMemberStatusActive,
		},
		{
			OrganizationID: 2, OrganizationName: "Suspended", OrganizationStatus: model.OrganizationStatusSuspended,
			UserID: 10, Role: model.OrganizationRoleOwner, Status: model.OrganizationMemberStatusActive,
		},
		{
			OrganizationID: 3, OrganizationName: "Revoked", OrganizationStatus: model.OrganizationStatusActive,
			UserID: 10, Role: model.OrganizationRoleOwner, Status: model.OrganizationMemberStatusRevoked,
		},
	}}
	service := NewOrganizationAuthorizationService(repository)

	allowed, err := service.Authorize(10, 1, authorization.CapabilityEventsManage)
	if err != nil || allowed == nil || allowed.Role != model.OrganizationRoleEditor {
		t.Fatalf("expected editor event access: context=%+v err=%v", allowed, err)
	}
	for _, test := range []struct {
		name         string
		userID       int64
		organization int64
		capability   authorization.Capability
	}{
		{"editor member management", 10, 1, authorization.CapabilityMembersManage},
		{"cross tenant", 11, 1, authorization.CapabilityOrganizationRead},
		{"suspended organization", 10, 2, authorization.CapabilityOrganizationRead},
		{"revoked membership", 10, 3, authorization.CapabilityOrganizationRead},
		{"invalid organization", 10, 0, authorization.CapabilityOrganizationRead},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := service.Authorize(test.userID, test.organization, test.capability); !errors.Is(err, authorization.ErrOrganizationAccessDenied) {
				t.Fatalf("expected access denied, got %v", err)
			}
		})
	}
}

func TestOrganizationAuthorizationServiceListsInactiveContextWithoutCapabilities(t *testing.T) {
	repository := &organizationAuthorizationRepositoryStub{memberships: []*model.OrganizationMember{
		{
			OrganizationID: 1, OrganizationStatus: model.OrganizationStatusActive,
			UserID: 10, Role: model.OrganizationRoleFinance, Status: model.OrganizationMemberStatusActive,
		},
		{
			OrganizationID: 2, OrganizationStatus: model.OrganizationStatusSuspended,
			UserID: 10, Role: model.OrganizationRoleOwner, Status: model.OrganizationMemberStatusActive,
		},
	}}
	contexts, err := NewOrganizationAuthorizationService(repository).ListForUser(10)
	if err != nil || len(contexts) != 2 {
		t.Fatalf("unexpected contexts: contexts=%+v err=%v", contexts, err)
	}
	if len(contexts[0].Capabilities) == 0 {
		t.Fatal("active finance membership must expose capabilities")
	}
	if len(contexts[1].Capabilities) != 0 {
		t.Fatalf("suspended organization exposed capabilities: %v", contexts[1].Capabilities)
	}
}
