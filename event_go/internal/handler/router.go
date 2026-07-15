package handler

import (
	"net/http"

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
		{http.MethodGet, "/me/registrations", "listMyRegistrations", "", plain(h.ListMyRegistrations)},
		{http.MethodGet, "/admin/identity-migration", "getIdentityMigrationReport", "", admin(h.GetIdentityMigrationReport)},

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

		{http.MethodPost, "/events/{id}/posts", "createPost", "CreatePostRequest", plain(h.CreatePost)},
		{http.MethodGet, "/events/{id}/posts", "listPosts", "", plain(h.ListPosts)},
		{http.MethodGet, "/events/{id}/posts/{postId}", "getPost", "", plain(h.GetPost)},
		{http.MethodPost, "/events/{id}/posts/{postId}/replies", "createReply", "CreateReplyRequest", plain(h.CreateReply)},

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
	mux.HandleFunc("GET /health", h.HealthHandler)
	mux.Handle("/", fallback)

	return LoggingMiddleware(SecurityHeaders(CORS(UserAuth(mux, h.tokens), h.config.CORSOrigin)))
}
