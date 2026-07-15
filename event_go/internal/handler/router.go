package handler

import (
	"net/http"
	"sort"
	"strings"

	"github.com/qw2261/soulmarker/event_go/internal/api"
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
	admin := func(handler http.HandlerFunc) http.Handler {
		return AdminAuth(handler, h.config.AdminToken)
	}
	plain := func(handler http.HandlerFunc) http.Handler {
		return handler
	}

	return []apiRoute{
		{http.MethodPost, "/auth/register", "registerUser", "RegisterUserRequest", plain(h.RegisterUser)},
		{http.MethodPost, "/auth/login", "loginUser", "LoginRequest", plain(h.Login)},
		{http.MethodPost, "/auth/logout", "logoutUser", "", plain(h.Logout)},
		{http.MethodPost, "/auth/password-reset/request", "requestPasswordReset", "PasswordResetRequest", plain(h.RequestPasswordReset)},
		{http.MethodPost, "/auth/password-reset/confirm", "confirmPasswordReset", "PasswordResetConfirmRequest", plain(h.ConfirmPasswordReset)},
		{http.MethodGet, "/me/registrations", "listMyRegistrations", "", plain(h.ListMyRegistrations)},
		{http.MethodGet, "/me/admissions", "listMyAdmissions", "", plain(h.ListMyAdmissions)},
		{http.MethodGet, "/me/activities", "listMyActivities", "", plain(h.ListMyActivities)},
		{http.MethodGet, "/admin/session", "getAdminSession", "", admin(h.GetAdminSession)},
		{http.MethodGet, "/admin/identity-migration", "getIdentityMigrationReport", "", admin(h.GetIdentityMigrationReport)},
		{http.MethodGet, "/admin/content-reports", "listContentReports", "", admin(h.ListContentReports)},
		{http.MethodPut, "/admin/content-reports/{reportId}", "resolveContentReport", "ResolveContentReportRequest", admin(h.ResolveContentReport)},
		{http.MethodGet, "/admin/content-actions", "listContentModerationActions", "", admin(h.ListContentModerationActions)},

		{http.MethodPost, "/organizers", "createOrganizer", "CreateOrganizerRequest", admin(h.CreateOrganizer)},
		{http.MethodGet, "/organizers", "listOrganizers", "", plain(h.ListOrganizers)},
		{http.MethodGet, "/organizers/{id}", "getOrganizer", "", plain(h.GetOrganizer)},
		{http.MethodPut, "/organizers/{id}", "updateOrganizer", "UpdateOrganizerRequest", admin(h.UpdateOrganizer)},
		{http.MethodDelete, "/organizers/{id}", "deleteOrganizer", "", admin(h.DeleteOrganizer)},

		{http.MethodPost, "/events", "createEvent", "CreateEventRequest", admin(h.CreateEvent)},
		{http.MethodGet, "/events", "listEvents", "", plain(h.ListEvents)},
		{http.MethodGet, "/events/{id}", "getEvent", "", plain(h.GetEvent)},
		{http.MethodPut, "/events/{id}", "updateEvent", "UpdateEventRequest", admin(h.UpdateEvent)},
		{http.MethodDelete, "/events/{id}", "deleteEvent", "", admin(h.DeleteEvent)},

		{http.MethodPost, "/events/{id}/register", "registerForEvent", "RegisterEventRequest", plain(h.Register)},
		{http.MethodDelete, "/events/{id}/register", "cancelEventRegistration", "", plain(h.CancelRegistration)},
		{http.MethodGet, "/events/{id}/registration", "getEventRegistrationStatus", "", plain(h.GetRegistrationStatus)},
		{http.MethodGet, "/events/{id}/registrations", "listEventRegistrations", "", admin(h.ListRegistrations)},
		{http.MethodGet, "/events/{id}/admission", "getMyAdmission", "", plain(h.GetMyAdmission)},
		{http.MethodPost, "/events/{id}/checkins", "checkInAdmission", "CheckinRequest", admin(h.CheckIn)},
		{http.MethodGet, "/events/{id}/checkins", "listEventCheckins", "", admin(h.ListCheckins)},

		{http.MethodPost, "/events/{id}/posts", "createPost", "CreatePostRequest", plain(h.CreatePost)},
		{http.MethodGet, "/events/{id}/posts", "listPosts", "", plain(h.ListPosts)},
		{http.MethodGet, "/events/{id}/posts/{postId}", "getPost", "", plain(h.GetPost)},
		{http.MethodPost, "/events/{id}/posts/{postId}/reports", "reportPost", "CreateContentReportRequest", plain(h.ReportPost)},
		{http.MethodDelete, "/events/{id}/posts/{postId}", "removePost", "ModerateContentRequest", admin(h.RemovePost)},
		{http.MethodPut, "/events/{id}/posts/{postId}/restore", "restorePost", "ModerateContentRequest", admin(h.RestorePost)},
		{http.MethodPost, "/events/{id}/posts/{postId}/replies", "createReply", "CreateReplyRequest", plain(h.CreateReply)},
		{http.MethodPost, "/events/{id}/posts/{postId}/replies/{replyId}/reports", "reportReply", "CreateContentReportRequest", plain(h.ReportReply)},
		{http.MethodDelete, "/events/{id}/posts/{postId}/replies/{replyId}", "removeReply", "ModerateContentRequest", admin(h.RemoveReply)},
		{http.MethodPut, "/events/{id}/posts/{postId}/replies/{replyId}/restore", "restoreReply", "ModerateContentRequest", admin(h.RestoreReply)},

		{http.MethodPost, "/events/{id}/tickets", "createTicket", "CreateTicketRequest", admin(h.CreateTicket)},
		{http.MethodGet, "/events/{id}/tickets", "listTickets", "", plain(h.ListTickets)},
		{http.MethodGet, "/events/{id}/tickets/{ticketId}", "getTicket", "", plain(h.GetTicket)},
		{http.MethodPut, "/events/{id}/tickets/{ticketId}", "updateTicket", "UpdateTicketRequest", admin(h.UpdateTicket)},
		{http.MethodDelete, "/events/{id}/tickets/{ticketId}", "deleteTicket", "", admin(h.DeleteTicket)},
	}
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

	return LoggingMiddleware(SecurityHeaders(CORS(UserAuth(mux, h.tokens), h.config.CORSOrigin)))
}
