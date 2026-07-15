package authorization

import (
	"context"
	"reflect"
	"testing"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func TestRoleCapabilityMatrixUsesLeastPrivilege(t *testing.T) {
	want := map[string][]Capability{
		model.OrganizationRoleOwner: AllCapabilities(),
		model.OrganizationRoleAdmin: {
			CapabilityAuditRead, CapabilityCheckinsManage, CapabilityEventsManage,
			CapabilityMembersInvite, CapabilityMembersManage, CapabilityMembersRead,
			CapabilityOrganizationManage, CapabilityOrganizationRead,
			CapabilityRegistrationsRead, CapabilityTicketsManage,
		},
		model.OrganizationRoleEditor: {
			CapabilityEventsManage, CapabilityOrganizationManage,
			CapabilityOrganizationRead, CapabilityTicketsManage,
		},
		model.OrganizationRoleChecker: {
			CapabilityCheckinsManage, CapabilityOrganizationRead,
		},
		model.OrganizationRoleFinance: {
			CapabilityFinanceRead, CapabilityOrganizationRead,
			CapabilityRegistrationsExport, CapabilityRegistrationsRead,
		},
	}
	for role, expected := range want {
		actual := CapabilitiesForRole(role)
		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("capabilities for %s: got=%v want=%v", role, actual, expected)
		}
		for _, capability := range AllCapabilities() {
			if Allows(role, capability) != containsCapability(expected, capability) {
				t.Fatalf("allows mismatch: role=%s capability=%s", role, capability)
			}
		}
	}
	if capabilities := CapabilitiesForRole("platform_admin"); len(capabilities) != 0 {
		t.Fatalf("platform admin must not inherit tenant capabilities: %v", capabilities)
	}
	if Allows("unknown", CapabilityOrganizationRead) {
		t.Fatal("unknown role received a tenant capability")
	}
}

func TestPrincipalContextsAreSeparated(t *testing.T) {
	organization := OrganizationContext{OrganizationID: 7, UserID: 9, Role: model.OrganizationRoleOwner}
	ctx := WithOrganizationContext(context.Background(), organization)
	if IsPlatformAdmin(ctx) {
		t.Fatal("tenant principal was treated as platform admin")
	}
	if got, ok := OrganizationContextFromContext(ctx); !ok || got.OrganizationID != organization.OrganizationID {
		t.Fatalf("organization context missing: got=%+v ok=%v", got, ok)
	}

	platformContext := WithPlatformAdmin(context.Background())
	if !IsPlatformAdmin(platformContext) {
		t.Fatal("platform admin principal missing")
	}
	if _, ok := OrganizationContextFromContext(platformContext); ok {
		t.Fatal("platform admin inherited an organization context")
	}
}

func containsCapability(values []Capability, target Capability) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
