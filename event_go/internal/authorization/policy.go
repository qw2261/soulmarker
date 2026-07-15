package authorization

import (
	"context"
	"errors"
	"sort"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

type Capability string

const (
	CapabilityOrganizationRead      Capability = "organization.read"
	CapabilityOrganizationManage    Capability = "organization.manage"
	CapabilityMembersRead           Capability = "members.read"
	CapabilityMembersInvite         Capability = "members.invite"
	CapabilityMembersManage         Capability = "members.manage"
	CapabilityOwnerTransfer         Capability = "owner.transfer"
	CapabilityEventsManage          Capability = "events.manage"
	CapabilityTicketsManage         Capability = "tickets.manage"
	CapabilityRegistrationsRead     Capability = "registrations.read"
	CapabilityRegistrationsExport   Capability = "registrations.export"
	CapabilityCheckinsManage        Capability = "checkins.manage"
	CapabilityFinanceRead           Capability = "finance.read"
	CapabilityAuditRead             Capability = "audit.read"
	PrincipalTypePlatformAdmin                 = "platform_admin"
	PrincipalTypeOrganizationMember            = "organization_member"
)

var ErrOrganizationAccessDenied = errors.New("无权访问该组织")

var allCapabilities = []Capability{
	CapabilityOrganizationRead,
	CapabilityOrganizationManage,
	CapabilityMembersRead,
	CapabilityMembersInvite,
	CapabilityMembersManage,
	CapabilityOwnerTransfer,
	CapabilityEventsManage,
	CapabilityTicketsManage,
	CapabilityRegistrationsRead,
	CapabilityRegistrationsExport,
	CapabilityCheckinsManage,
	CapabilityFinanceRead,
	CapabilityAuditRead,
}

var roleCapabilities = map[string][]Capability{
	model.OrganizationRoleOwner: append([]Capability(nil), allCapabilities...),
	model.OrganizationRoleAdmin: {
		CapabilityOrganizationRead,
		CapabilityOrganizationManage,
		CapabilityMembersRead,
		CapabilityMembersInvite,
		CapabilityMembersManage,
		CapabilityEventsManage,
		CapabilityTicketsManage,
		CapabilityRegistrationsRead,
		CapabilityCheckinsManage,
		CapabilityAuditRead,
	},
	model.OrganizationRoleEditor: {
		CapabilityOrganizationRead,
		CapabilityOrganizationManage,
		CapabilityEventsManage,
		CapabilityTicketsManage,
	},
	model.OrganizationRoleChecker: {
		CapabilityOrganizationRead,
		CapabilityCheckinsManage,
	},
	model.OrganizationRoleFinance: {
		CapabilityOrganizationRead,
		CapabilityRegistrationsRead,
		CapabilityRegistrationsExport,
		CapabilityFinanceRead,
	},
}

func AllCapabilities() []Capability {
	result := append([]Capability(nil), allCapabilities...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func CapabilitiesForRole(role string) []Capability {
	result := append([]Capability(nil), roleCapabilities[role]...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func Allows(role string, capability Capability) bool {
	for _, allowed := range roleCapabilities[role] {
		if allowed == capability {
			return true
		}
	}
	return false
}

type OrganizationContext struct {
	OrganizationID     int64
	OrganizationName   string
	OrganizationSlug   string
	OrganizationStatus string
	MembershipStatus   string
	UserID             int64
	Role               string
	Capabilities       []Capability
}

type contextKey uint8

const (
	organizationContextKey contextKey = iota
	platformAdminContextKey
)

func WithOrganizationContext(ctx context.Context, value OrganizationContext) context.Context {
	return context.WithValue(ctx, organizationContextKey, value)
}

func OrganizationContextFromContext(ctx context.Context) (OrganizationContext, bool) {
	value, ok := ctx.Value(organizationContextKey).(OrganizationContext)
	return value, ok
}

func WithPlatformAdmin(ctx context.Context) context.Context {
	return context.WithValue(ctx, platformAdminContextKey, true)
}

func IsPlatformAdmin(ctx context.Context) bool {
	value, _ := ctx.Value(platformAdminContextKey).(bool)
	return value
}
