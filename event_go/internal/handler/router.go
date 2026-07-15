package handler

import "net/http"

// NewRouter 创建生产与集成测试共用的唯一路由和中间件组合。
func NewRouter(h *Handler, fallback http.Handler) http.Handler {
	if fallback == nil {
		fallback = http.NotFoundHandler()
	}

	mux := http.NewServeMux()
	adminAuth := func(next http.Handler) http.Handler {
		return AdminAuth(next, h.config.AdminToken)
	}

	mux.HandleFunc("POST /api/auth/register", h.RegisterUser)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("GET /api/me/registrations", h.ListMyRegistrations)
	mux.HandleFunc("GET /api/admin/identity-migration", adminAuth(http.HandlerFunc(h.GetIdentityMigrationReport)).ServeHTTP)

	mux.HandleFunc("POST /api/organizers", adminAuth(http.HandlerFunc(h.CreateOrganizer)).ServeHTTP)
	mux.HandleFunc("GET /api/organizers", h.ListOrganizers)
	mux.HandleFunc("GET /api/organizers/{id}", h.GetOrganizer)
	mux.HandleFunc("PUT /api/organizers/{id}", adminAuth(http.HandlerFunc(h.UpdateOrganizer)).ServeHTTP)
	mux.HandleFunc("DELETE /api/organizers/{id}", adminAuth(http.HandlerFunc(h.DeleteOrganizer)).ServeHTTP)

	mux.HandleFunc("POST /api/events", adminAuth(http.HandlerFunc(h.CreateEvent)).ServeHTTP)
	mux.HandleFunc("GET /api/events", h.ListEvents)
	mux.HandleFunc("GET /api/events/{id}", h.GetEvent)
	mux.HandleFunc("PUT /api/events/{id}", adminAuth(http.HandlerFunc(h.UpdateEvent)).ServeHTTP)
	mux.HandleFunc("DELETE /api/events/{id}", adminAuth(http.HandlerFunc(h.DeleteEvent)).ServeHTTP)

	mux.HandleFunc("POST /api/events/{id}/register", h.Register)
	mux.HandleFunc("DELETE /api/events/{id}/register", h.CancelRegistration)
	mux.HandleFunc("GET /api/events/{id}/registration", h.GetRegistrationStatus)
	mux.HandleFunc("GET /api/events/{id}/registrations", adminAuth(http.HandlerFunc(h.ListRegistrations)).ServeHTTP)

	mux.HandleFunc("POST /api/events/{id}/posts", h.CreatePost)
	mux.HandleFunc("GET /api/events/{id}/posts", h.ListPosts)
	mux.HandleFunc("GET /api/events/{id}/posts/{postId}", h.GetPost)
	mux.HandleFunc("POST /api/events/{id}/posts/{postId}/replies", h.CreateReply)

	mux.HandleFunc("POST /api/events/{id}/tickets", adminAuth(http.HandlerFunc(h.CreateTicket)).ServeHTTP)
	mux.HandleFunc("GET /api/events/{id}/tickets", h.ListTickets)
	mux.HandleFunc("GET /api/events/{id}/tickets/{ticketId}", h.GetTicket)
	mux.HandleFunc("PUT /api/events/{id}/tickets/{ticketId}", adminAuth(http.HandlerFunc(h.UpdateTicket)).ServeHTTP)
	mux.HandleFunc("DELETE /api/events/{id}/tickets/{ticketId}", adminAuth(http.HandlerFunc(h.DeleteTicket)).ServeHTTP)

	mux.HandleFunc("GET /health", h.HealthHandler)
	mux.Handle("/", fallback)

	return LoggingMiddleware(SecurityHeaders(CORS(UserAuth(mux, h.config.JWTSecret), h.config.CORSOrigin)))
}
