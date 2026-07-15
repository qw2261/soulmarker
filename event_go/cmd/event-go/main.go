package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/config"
	"github.com/qw2261/soulmarker/event_go/internal/handler"
	"github.com/qw2261/soulmarker/event_go/internal/store"
)

const staticDir = "web/dist"

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("配置校验失败: %v", err)
	}
	handler.ConfigureLogging(cfg)

	s, err := store.OpenStore(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer s.Close()
	log.Printf("📦 数据库已初始化: %s", cfg.DatabasePath)

	h := handler.NewHandler(s, cfg)

	port := cfg.Port
	addr := ":" + port
	server := &http.Server{
		Addr:              addr,
		Handler:           handler.NewRouter(h, spaHandler(staticDir)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("亦闻 event-go 服务启动，监听端口 %s", port)
	log.Printf("API 文档：")
	log.Printf("  GET    /health                               健康检查")
	log.Printf("  POST   /api/auth/register                    用户注册")
	log.Printf("  POST   /api/auth/login                       用户登录")
	log.Printf("  GET    /api/me/registrations                 当前用户报名列表")
	log.Printf("  GET    /api/admin/identity-migration         身份迁移报告 🔐")
	log.Printf("  POST   /api/organizers                       创建门店 🔐")
	log.Printf("  GET    /api/organizers                       门店列表")
	log.Printf("  GET    /api/organizers/{id}                  门店详情")
	log.Printf("  PUT    /api/organizers/{id}                  编辑门店 🔐")
	log.Printf("  DELETE /api/organizers/{id}                  删除门店 🔐")
	log.Printf("  POST   /api/events                          创建活动 🔐")
	log.Printf("  GET    /api/events                          活动列表")
	log.Printf("  GET    /api/events/{id}                     活动详情")
	log.Printf("  PUT    /api/events/{id}                     编辑活动 🔐")
	log.Printf("  DELETE /api/events/{id}                     删除活动 🔐")
	log.Printf("  POST   /api/events/{id}/register            报名活动")
	log.Printf("  DELETE /api/events/{id}/register            取消报名")
	log.Printf("  GET    /api/events/{id}/registration        当前用户报名状态")
	log.Printf("  GET    /api/events/{id}/registrations       报名列表 🔐")
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

func spaHandler(root string) http.HandlerFunc {
	fs := http.FileServer(http.Dir(root))

	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		fullPath := root + path

		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			r.URL.Path = "/"
		}

		fs.ServeHTTP(w, r)
	}
}
