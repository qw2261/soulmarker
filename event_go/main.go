package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = defaultDB
	}

	store := NewStore(dbPath)
	defer store.Close()
	log.Printf("📦 数据库已初始化: %s", dbPath)

	h := NewHandler(store)
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/events", adminAuth(http.HandlerFunc(h.CreateEvent)).ServeHTTP)
	mux.HandleFunc("GET /api/events", h.ListEvents)
	mux.HandleFunc("GET /api/events/{id}", h.GetEvent)
	mux.HandleFunc("PUT /api/events/{id}", adminAuth(http.HandlerFunc(h.UpdateEvent)).ServeHTTP)
	mux.HandleFunc("DELETE /api/events/{id}", adminAuth(http.HandlerFunc(h.DeleteEvent)).ServeHTTP)
	mux.HandleFunc("POST /api/events/{id}/register", h.Register)
	mux.HandleFunc("GET /api/events/{id}/registrations", h.ListRegistrations)
	mux.HandleFunc("POST /api/events/{id}/posts", h.CreatePost)
	mux.HandleFunc("GET /api/events/{id}/posts", h.ListPosts)
	mux.HandleFunc("GET /api/events/{id}/posts/{postId}", h.GetPost)
	mux.HandleFunc("POST /api/events/{id}/posts/{postId}/replies", h.CreateReply)

	mux.HandleFunc("POST /api/events/{id}/tickets", adminAuth(http.HandlerFunc(h.CreateTicket)).ServeHTTP)
	mux.HandleFunc("GET /api/events/{id}/tickets", h.ListTickets)
	mux.HandleFunc("GET /api/events/{id}/tickets/{ticketId}", h.GetTicket)
	mux.HandleFunc("PUT /api/events/{id}/tickets/{ticketId}", adminAuth(http.HandlerFunc(h.UpdateTicket)).ServeHTTP)
	mux.HandleFunc("DELETE /api/events/{id}/tickets/{ticketId}", adminAuth(http.HandlerFunc(h.DeleteTicket)).ServeHTTP)

	port := getPort()
	addr := ":" + port
	server := &http.Server{
		Addr:    addr,
		Handler: cors(mux),
	}

	log.Printf("亦闻 event-go 服务启动，监听端口 %s", port)
	log.Printf("API 文档：")
	log.Printf("  POST   /api/events                          创建活动 🔐")
	log.Printf("  GET    /api/events                          活动列表")
	log.Printf("  GET    /api/events/{id}                     活动详情")
	log.Printf("  PUT    /api/events/{id}                     编辑活动 🔐")
	log.Printf("  DELETE /api/events/{id}                     删除活动 🔐")
	log.Printf("  POST   /api/events/{id}/register            报名活动")
	log.Printf("  GET    /api/events/{id}/registrations       报名列表")
	log.Printf("  POST   /api/events/{id}/posts               发帖（需已报名）")
	log.Printf("  GET    /api/events/{id}/posts               帖子列表")
	log.Printf("  GET    /api/events/{id}/posts/{postId}      帖子详情（含回复）")
	log.Printf("  POST   /api/events/{id}/posts/{postId}/replies 回复帖子（需已报名）")
	log.Printf("  POST   /api/events/{id}/tickets             创建门票 🔐")
	log.Printf("  GET    /api/events/{id}/tickets             门票列表")
	log.Printf("  GET    /api/events/{id}/tickets/{ticketId}  门票详情")
	log.Printf("  PUT    /api/events/{id}/tickets/{ticketId}  编辑门票 🔐")
	log.Printf("  DELETE /api/events/{id}/tickets/{ticketId}  删除门票 🔐")
	log.Printf("")
	log.Printf("  🔐 = 需设置 ADMIN_TOKEN 环境变量进行认证")

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 正在关闭服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("服务关闭失败: %v", err)
	}
	log.Println("✅ 服务已关闭")
}
