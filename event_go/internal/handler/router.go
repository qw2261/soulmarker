package handler

import "net/http"

// NewRouter 创建生产与集成测试共用的唯一路由和中间件组合。
func NewRouter(h *Handler, fallback http.Handler) http.Handler {
	if fallback == nil {
		fallback = http.NotFoundHandler()
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/auth/register", h.RegisterUser)
	mux.HandleFunc("POST /api/auth/login", h.Login)

	mux.HandleFunc("POST /api/organizers", AdminAuth(http.HandlerFunc(h.CreateOrganizer)).ServeHTTP)
	mux.HandleFunc("GET /api/organizers", h.ListOrganizers)
	mux.HandleFunc("GET /api/organizers/{id}", h.GetOrganizer)
	mux.HandleFunc("PUT /api/organizers/{id}", AdminAuth(http.HandlerFunc(h.UpdateOrganizer)).ServeHTTP)
	mux.HandleFunc("DELETE /api/organizers/{id}", AdminAuth(http.HandlerFunc(h.DeleteOrganizer)).ServeHTTP)

	mux.HandleFunc("POST /api/events", AdminAuth(http.HandlerFunc(h.CreateEvent)).ServeHTTP)
	mux.HandleFunc("GET /api/events", h.ListEvents)
	mux.HandleFunc("GET /api/events/{id}", h.GetEvent)
	mux.HandleFunc("PUT /api/events/{id}", AdminAuth(http.HandlerFunc(h.UpdateEvent)).ServeHTTP)
	mux.HandleFunc("DELETE /api/events/{id}", AdminAuth(http.HandlerFunc(h.DeleteEvent)).ServeHTTP)

	mux.HandleFunc("POST /api/events/{id}/register", h.Register)
	mux.HandleFunc("DELETE /api/events/{id}/register", h.CancelRegistration)
	mux.HandleFunc("GET /api/events/{id}/registration", h.GetRegistrationStatus)
	mux.HandleFunc("GET /api/events/{id}/registrations", AdminAuth(http.HandlerFunc(h.ListRegistrations)).ServeHTTP)

	mux.HandleFunc("POST /api/events/{id}/posts", h.CreatePost)
	mux.HandleFunc("GET /api/events/{id}/posts", h.ListPosts)
	mux.HandleFunc("GET /api/events/{id}/posts/{postId}", h.GetPost)
	mux.HandleFunc("POST /api/events/{id}/posts/{postId}/replies", h.CreateReply)

	mux.HandleFunc("POST /api/events/{id}/tickets", AdminAuth(http.HandlerFunc(h.CreateTicket)).ServeHTTP)
	mux.HandleFunc("GET /api/events/{id}/tickets", h.ListTickets)
	mux.HandleFunc("GET /api/events/{id}/tickets/{ticketId}", h.GetTicket)
	mux.HandleFunc("PUT /api/events/{id}/tickets/{ticketId}", AdminAuth(http.HandlerFunc(h.UpdateTicket)).ServeHTTP)
	mux.HandleFunc("DELETE /api/events/{id}/tickets/{ticketId}", AdminAuth(http.HandlerFunc(h.DeleteTicket)).ServeHTTP)

	mux.HandleFunc("GET /health", h.HealthHandler)
	mux.Handle("/", fallback)

	return LoggingMiddleware(SecurityHeaders(CORS(UserAuth(mux))))
}
