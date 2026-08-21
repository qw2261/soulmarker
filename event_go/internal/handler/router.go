package handler

import (
	"net/http"
	"sort"
	"strings"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/authorization"
	"github.com/qw2261/soulmarker/event_go/internal/openapi"
)

type apiRoute struct {
	Method        string
	Path          string
	OperationID   string
	RequestSchema string
	Handler       http.Handler
}

func (h *Handler) apiRoutes() []apiRoute {
	platformAdmin := func(handler http.HandlerFunc) http.Handler {
		return PlatformAdminAuth(handler, h.config.AdminToken)
	}
	tenant := func(capability authorization.Capability, handler http.HandlerFunc) http.Handler {
		return h.OrganizationAuth(handler, capability)
	}
	plain := func(handler http.HandlerFunc) http.Handler {
		return handler
	}

	routes := []apiRoute{
		{http.MethodPost, "/auth/register", "registerUser", "RegisterUserRequest", plain(h.RegisterUser)},
		{http.MethodPost, "/auth/login", "loginUser", "LoginRequest", plain(h.Login)},
		{http.MethodPost, "/auth/logout", "logoutUser", "", plain(h.Logout)},
		{http.MethodPost, "/auth/password-reset/request", "requestPasswordReset", "PasswordResetRequest", plain(h.RequestPasswordReset)},
		{http.MethodPost, "/auth/password-reset/confirm", "confirmPasswordReset", "PasswordResetConfirmRequest", plain(h.ConfirmPasswordReset)},
		{http.MethodPost, "/auth/recovery-email/confirm", "confirmRecoveryEmail", "RecoveryEmailConfirmRequest", plain(h.ConfirmRecoveryEmail)},
		{http.MethodGet, "/me/registrations", "listMyRegistrations", "", plain(h.ListMyRegistrations)},
		{http.MethodGet, "/me/admissions", "listMyAdmissions", "", plain(h.ListMyAdmissions)},
		{http.MethodGet, "/me/activities", "listMyActivities", "", plain(h.ListMyActivities)},
		{http.MethodPost, "/me/recovery-email/request", "requestRecoveryEmail", "RecoveryEmailRequest", plain(h.RequestRecoveryEmail)},
		{http.MethodGet, "/me/notifications", "listNotifications", "", plain(h.ListNotifications)},
		{http.MethodGet, "/me/notifications/unread-count", "getNotificationUnreadCount", "", plain(h.GetNotificationUnreadCount)},
		{http.MethodPut, "/me/notifications/{notificationId}/read", "markNotificationRead", "", plain(h.MarkNotificationRead)},
		{http.MethodPut, "/me/notifications/read-all", "markAllNotificationsRead", "", plain(h.MarkAllNotificationsRead)},
		{http.MethodGet, "/me/organizations", "listMyOrganizations", "", plain(h.ListMyOrganizations)},
		{http.MethodPost, "/organizations", "createOrganization", "CreateOrganizationRequest", plain(h.CreateOrganization)},
		{http.MethodPost, "/organization-invitations/accept", "acceptOrganizationInvitation", "AcceptOrganizationInvitationRequest", plain(h.AcceptOrganizationInvitation)},
		{http.MethodGet, "/organizations/{organizationId}/session", "getOrganizationSession", "", tenant(authorization.CapabilityOrganizationRead, h.GetOrganizationSession)},
		{http.MethodGet, "/organizations/{organizationId}", "getOrganizationWorkspace", "", tenant(authorization.CapabilityOrganizationRead, h.GetOrganizationWorkspace)},
		{http.MethodGet, "/organizations/{organizationId}/members", "listOrganizationMembers", "", tenant(authorization.CapabilityMembersRead, h.ListOrganizationMembers)},
		{http.MethodPut, "/organizations/{organizationId}/members/{memberId}", "updateOrganizationMember", "UpdateOrganizationMemberRequest", tenant(authorization.CapabilityMembersManage, h.UpdateOrganizationMember)},
		{http.MethodDelete, "/organizations/{organizationId}/members/{memberId}", "revokeOrganizationMember", "", tenant(authorization.CapabilityMembersManage, h.RevokeOrganizationMember)},
		{http.MethodPost, "/organizations/{organizationId}/owner-transfer", "transferOrganizationOwnership", "TransferOrganizationOwnerRequest", tenant(authorization.CapabilityOwnerTransfer, h.TransferOrganizationOwnership)},
		{http.MethodGet, "/organizations/{organizationId}/invitations", "listOrganizationInvitations", "", tenant(authorization.CapabilityMembersRead, h.ListOrganizationInvitations)},
		{http.MethodPost, "/organizations/{organizationId}/invitations", "createOrganizationInvitation", "CreateOrganizationInvitationRequest", tenant(authorization.CapabilityMembersInvite, h.CreateOrganizationInvitation)},
		{http.MethodDelete, "/organizations/{organizationId}/invitations/{invitationId}", "revokeOrganizationInvitation", "", tenant(authorization.CapabilityMembersInvite, h.RevokeOrganizationInvitation)},
		{http.MethodGet, "/organizations/{organizationId}/audits", "listOrganizationAudits", "", tenant(authorization.CapabilityAuditRead, h.ListOrganizationAudits)},
		{http.MethodPost, "/organizations/{organizationId}/events", "createOrganizationEvent", "CreateEventRequest", tenant(authorization.CapabilityEventsManage, h.CreateEvent)},
		{http.MethodGet, "/organizations/{organizationId}/events", "listOrganizationEvents", "", tenant(authorization.CapabilityOrganizationRead, h.ListOrganizationEvents)},
		{http.MethodGet, "/organizations/{organizationId}/events/{id}", "getOrganizationEvent", "", tenant(authorization.CapabilityOrganizationRead, h.GetOrganizationEvent)},
		{http.MethodPut, "/organizations/{organizationId}/events/{id}", "updateOrganizationEvent", "UpdateEventRequest", tenant(authorization.CapabilityEventsManage, h.UpdateEvent)},
		{http.MethodDelete, "/organizations/{organizationId}/events/{id}", "deleteOrganizationEvent", "", tenant(authorization.CapabilityEventsManage, h.DeleteEvent)},
		{http.MethodPost, "/organizations/{organizationId}/events/{id}/tickets", "createOrganizationTicket", "CreateTicketRequest", tenant(authorization.CapabilityTicketsManage, h.CreateTicket)},
		{http.MethodGet, "/organizations/{organizationId}/events/{id}/tickets", "listOrganizationTickets", "", tenant(authorization.CapabilityOrganizationRead, h.ListOrganizationTickets)},
		{http.MethodGet, "/organizations/{organizationId}/events/{id}/tickets/{ticketId}", "getOrganizationTicket", "", tenant(authorization.CapabilityOrganizationRead, h.GetOrganizationTicket)},
		{http.MethodPut, "/organizations/{organizationId}/events/{id}/tickets/{ticketId}", "updateOrganizationTicket", "UpdateTicketRequest", tenant(authorization.CapabilityTicketsManage, h.UpdateTicket)},
		{http.MethodDelete, "/organizations/{organizationId}/events/{id}/tickets/{ticketId}", "deleteOrganizationTicket", "", tenant(authorization.CapabilityTicketsManage, h.DeleteTicket)},
		{http.MethodGet, "/organizations/{organizationId}/events/{id}/registrations", "listOrganizationEventRegistrations", "", tenant(authorization.CapabilityRegistrationsRead, h.ListRegistrations)},
		{http.MethodGet, "/organizations/{organizationId}/events/{id}/registrations/export", "exportOrganizationEventRegistrations", "", tenant(authorization.CapabilityRegistrationsExport, h.ExportRegistrations)},
		{http.MethodPost, "/organizations/{organizationId}/events/{id}/checkins", "checkInOrganizationAdmission", "CheckinRequest", tenant(authorization.CapabilityCheckinsManage, h.CheckIn)},
		{http.MethodGet, "/organizations/{organizationId}/events/{id}/checkins", "listOrganizationEventCheckins", "", tenant(authorization.CapabilityCheckinsManage, h.ListCheckins)},
		{http.MethodGet, "/admin/session", "getAdminSession", "", platformAdmin(h.GetAdminSession)},
		{http.MethodGet, "/admin/identity-migration", "getIdentityMigrationReport", "", platformAdmin(h.GetIdentityMigrationReport)},
		{http.MethodGet, "/admin/content-reports", "listContentReports", "", platformAdmin(h.ListContentReports)},
		{http.MethodPut, "/admin/content-reports/{reportId}", "resolveContentReport", "ResolveContentReportRequest", platformAdmin(h.ResolveContentReport)},
		{http.MethodGet, "/admin/content-actions", "listContentModerationActions", "", platformAdmin(h.ListContentModerationActions)},

		{http.MethodPost, "/organizers", "createOrganizer", "CreateOrganizerRequest", platformAdmin(h.CreateOrganizer)},
		{http.MethodGet, "/organizers", "listOrganizers", "", plain(h.ListOrganizers)},
		{http.MethodGet, "/organizers/{id}", "getOrganizer", "", plain(h.GetOrganizer)},
		{http.MethodPut, "/organizers/{id}", "updateOrganizer", "UpdateOrganizerRequest", platformAdmin(h.UpdateOrganizer)},
		{http.MethodDelete, "/organizers/{id}", "deleteOrganizer", "", platformAdmin(h.DeleteOrganizer)},

		{http.MethodPost, "/events", "createEvent", "CreateEventRequest", platformAdmin(h.CreateEvent)},
		{http.MethodGet, "/events", "listEvents", "", plain(h.ListEvents)},
		{http.MethodGet, "/events/{id}", "getEvent", "", plain(h.GetEvent)},
		{http.MethodPut, "/events/{id}", "updateEvent", "UpdateEventRequest", platformAdmin(h.UpdateEvent)},
		{http.MethodDelete, "/events/{id}", "deleteEvent", "", platformAdmin(h.DeleteEvent)},

		{http.MethodPost, "/events/{id}/register", "registerForEvent", "RegisterEventRequest", plain(h.Register)},
		{http.MethodDelete, "/events/{id}/register", "cancelEventRegistration", "", plain(h.CancelRegistration)},
		{http.MethodGet, "/events/{id}/registration", "getEventRegistrationStatus", "", plain(h.GetRegistrationStatus)},
		{http.MethodGet, "/events/{id}/registrations", "listEventRegistrations", "", platformAdmin(h.ListRegistrations)},
		{http.MethodGet, "/events/{id}/registrations/export", "exportEventRegistrations", "", platformAdmin(h.ExportRegistrations)},
		{http.MethodGet, "/events/{id}/admission", "getMyAdmission", "", plain(h.GetMyAdmission)},
		{http.MethodPost, "/events/{id}/checkins", "checkInAdmission", "CheckinRequest", platformAdmin(h.CheckIn)},
		{http.MethodGet, "/events/{id}/checkins", "listEventCheckins", "", platformAdmin(h.ListCheckins)},

		{http.MethodPost, "/events/{id}/posts", "createPost", "CreatePostRequest", plain(h.CreatePost)},
		{http.MethodGet, "/events/{id}/posts", "listPosts", "", plain(h.ListPosts)},
		{http.MethodGet, "/events/{id}/posts/{postId}", "getPost", "", plain(h.GetPost)},
		{http.MethodPost, "/events/{id}/posts/{postId}/reports", "reportPost", "CreateContentReportRequest", plain(h.ReportPost)},
		{http.MethodDelete, "/events/{id}/posts/{postId}", "removePost", "ModerateContentRequest", platformAdmin(h.RemovePost)},
		{http.MethodPut, "/events/{id}/posts/{postId}/restore", "restorePost", "ModerateContentRequest", platformAdmin(h.RestorePost)},
		{http.MethodPost, "/events/{id}/posts/{postId}/replies", "createReply", "CreateReplyRequest", plain(h.CreateReply)},
		{http.MethodPost, "/events/{id}/posts/{postId}/replies/{replyId}/reports", "reportReply", "CreateContentReportRequest", plain(h.ReportReply)},
		{http.MethodDelete, "/events/{id}/posts/{postId}/replies/{replyId}", "removeReply", "ModerateContentRequest", platformAdmin(h.RemoveReply)},
		{http.MethodPut, "/events/{id}/posts/{postId}/replies/{replyId}/restore", "restoreReply", "ModerateContentRequest", platformAdmin(h.RestoreReply)},

		{http.MethodPost, "/events/{id}/tickets", "createTicket", "CreateTicketRequest", platformAdmin(h.CreateTicket)},
		{http.MethodGet, "/events/{id}/tickets", "listTickets", "", plain(h.ListTickets)},
		{http.MethodGet, "/events/{id}/tickets/{ticketId}", "getTicket", "", plain(h.GetTicket)},
		{http.MethodPut, "/events/{id}/tickets/{ticketId}", "updateTicket", "UpdateTicketRequest", platformAdmin(h.UpdateTicket)},
		{http.MethodDelete, "/events/{id}/tickets/{ticketId}", "deleteTicket", "", platformAdmin(h.DeleteTicket)},
	}
	for index := range routes {
		if strings.HasPrefix(routes[index].Path, "/organizations/{organizationId}") {
			routes[index].Handler = h.OrganizationAudit(
				routes[index].OperationID,
				auditResourceType(routes[index].Path),
				routes[index].Handler,
			)
		}
	}
	return routes
}

func registerAPIRoutes(mux *http.ServeMux, prefix string, routes []apiRoute) {
	for _, route := range routes {
		mux.Handle(route.Method+" "+prefix+route.Path, route.Handler)
	}
}

func routePathMatches(pattern, actual string) bool {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	actualParts := strings.Split(strings.Trim(actual, "/"), "/")
	if len(patternParts) != len(actualParts) {
		return false
	}
	for i := range patternParts {
		if strings.HasPrefix(patternParts[i], "{") && strings.HasSuffix(patternParts[i], "}") {
			if actualParts[i] == "" {
				return false
			}
			continue
		}
		if patternParts[i] != actualParts[i] {
			return false
		}
	}
	return true
}

func apiFallback(prefix string, routes []apiRoute) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, prefix)
		allowed := make([]string, 0)
		for _, route := range routes {
			if routePathMatches(route.Path, path) {
				allowed = append(allowed, route.Method)
			}
		}
		if len(allowed) == 0 {
			writeError(w, http.StatusNotFound, api.CodeAPIRouteNotFound, "")
			return
		}
		sort.Strings(allowed)
		w.Header().Set("Allow", strings.Join(allowed, ", "))
		writeError(w, http.StatusMethodNotAllowed, api.CodeMethodNotAllowed, "")
	})
}

func registerAPIFallback(mux *http.ServeMux, prefix string, routes []apiRoute) {
	fallback := apiFallback(prefix, routes)
	mux.Handle(prefix, fallback)
	mux.Handle(prefix+"/", fallback)
}

// NewRouter 创建生产与集成测试共用的唯一路由和中间件组合。
func NewRouter(h *Handler, fallback http.Handler) http.Handler {
	if fallback == nil {
		fallback = http.NotFoundHandler()
	}

	mux := http.NewServeMux()
	routes := h.apiRoutes()
	registerAPIRoutes(mux, "/api", routes)
	registerAPIRoutes(mux, "/api/v1", routes)
	mux.HandleFunc("GET /api/v1/openapi.json", openapi.ServeV1)
	v1FallbackRoutes := append([]apiRoute{{Method: http.MethodGet, Path: "/openapi.json"}}, routes...)
	registerAPIFallback(mux, "/api/v1", v1FallbackRoutes)
	registerAPIFallback(mux, "/api", routes)
	mux.HandleFunc("GET /health", h.HealthHandler)
	mux.Handle("/", fallback)

	return RequestIDMiddleware(LoggingMiddleware(SecurityHeaders(CORS(UserAuth(mux, h.tokens), h.config.CORSOrigin))))
}
