package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/auth"
	"github.com/qw2261/soulmarker/event_go/internal/clock"
	"github.com/qw2261/soulmarker/event_go/internal/config"
	"github.com/qw2261/soulmarker/event_go/internal/handler"
	"github.com/qw2261/soulmarker/event_go/internal/identifier"
	"github.com/qw2261/soulmarker/event_go/internal/notification"
	"github.com/qw2261/soulmarker/event_go/internal/service"
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

	businessClock := clock.System{}
	tokens := auth.NewJWTManager(cfg.JWTSecret)
	credentials := identifier.CryptoCredentialGenerator{}
	registrations := service.NewRegistrationService(s, businessClock, time.Duration(cfg.CancelDeadlineHours)*time.Hour, credentials)
	admissions := service.NewAdmissionService(s, businessClock)
	discussions := service.NewDiscussionService(s)
	moderation := service.NewContentModerationService(s, businessClock)
	var resetSender notification.AuthenticationEmailSender = notification.LogPasswordResetSender{}
	if cfg.SMTPHost != "" {
		resetSender = notification.NewSMTPPasswordResetSender(
			cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom,
		)
	}
	authentication := service.NewAuthenticationService(
		s, businessClock, identifier.CryptoResetTokenGenerator{}, resetSender,
		cfg.PublicBaseURL, time.Duration(cfg.PasswordResetTTLMin)*time.Minute,
		time.Duration(cfg.RecoveryEmailTTLMin)*time.Minute,
	)
	h := handler.NewHandler(s, cfg, handler.Dependencies{
		Clock:          businessClock,
		Tokens:         tokens,
		Registrations:  registrations,
		Admissions:     admissions,
		Discussions:    discussions,
		Authentication: authentication,
		Moderation:     moderation,
	})

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
	log.Printf("API v1 文档：http://localhost:%s/api/v1/openapi.json", port)
	log.Printf("稳定 API 前缀：/api/v1（/api 保留一个兼容周期）")
	log.Printf("  GET    /health                               健康检查")
	log.Printf("  POST   /api/v1/auth/register                    用户注册")
	log.Printf("  POST   /api/v1/auth/login                       用户登录")
	log.Printf("  POST   /api/v1/auth/logout                      退出并撤销用户会话")
	log.Printf("  POST   /api/v1/auth/password-reset/request      请求密码重置")
	log.Printf("  POST   /api/v1/auth/password-reset/confirm      确认密码重置")
	log.Printf("  POST   /api/v1/auth/recovery-email/confirm      确认恢复邮箱绑定")
	log.Printf("  GET    /api/v1/me/registrations                 当前用户报名列表")
	log.Printf("  GET    /api/v1/me/admissions                    当前用户入场凭证")
	log.Printf("  GET    /api/v1/me/activities                    当前用户统一活动时间线")
	log.Printf("  POST   /api/v1/me/recovery-email/request        请求绑定恢复邮箱")
	log.Printf("  GET    /api/v1/admin/session                    校验平台管理员身份 🔐")
	log.Printf("  GET    /api/v1/admin/identity-migration         身份迁移报告 🔐")
	log.Printf("  POST   /api/v1/organizers                       创建门店 🔐")
	log.Printf("  GET    /api/v1/organizers                       门店列表")
	log.Printf("  GET    /api/v1/organizers/{id}                  门店详情")
	log.Printf("  PUT    /api/v1/organizers/{id}                  编辑门店 🔐")
	log.Printf("  DELETE /api/v1/organizers/{id}                  删除门店 🔐")
	log.Printf("  POST   /api/v1/events                           创建活动 🔐")
	log.Printf("  GET    /api/v1/events                           活动列表")
	log.Printf("  GET    /api/v1/events/{id}                      活动详情")
	log.Printf("  PUT    /api/v1/events/{id}                      编辑活动 🔐")
	log.Printf("  DELETE /api/v1/events/{id}                      删除活动 🔐")
	log.Printf("  POST   /api/v1/events/{id}/register             报名活动")
	log.Printf("  DELETE /api/v1/events/{id}/register             取消报名")
	log.Printf("  GET    /api/v1/events/{id}/registration         当前用户报名状态")
	log.Printf("  GET    /api/v1/events/{id}/registrations        报名列表 🔐")
	log.Printf("  GET    /api/v1/events/{id}/admission            当前用户活动凭证")
	log.Printf("  POST   /api/v1/events/{id}/checkins             核销入场凭证 🔐")
	log.Printf("  GET    /api/v1/events/{id}/checkins             核销审计列表 🔐")
	log.Printf("  POST   /api/v1/events/{id}/posts                发帖（需已报名）")
	log.Printf("  GET    /api/v1/events/{id}/posts                帖子列表")
	log.Printf("  GET    /api/v1/events/{id}/posts/{postId}       帖子详情（含回复）")
	log.Printf("  POST   /api/v1/events/{id}/posts/{postId}/replies 回复帖子（需已报名）")
	log.Printf("  POST   /api/v1/events/{id}/tickets              创建门票 🔐")
	log.Printf("  GET    /api/v1/events/{id}/tickets              门票列表")
	log.Printf("  GET    /api/v1/events/{id}/tickets/{ticketId}   门票详情")
	log.Printf("  PUT    /api/v1/events/{id}/tickets/{ticketId}   编辑门票 🔐")
	log.Printf("  DELETE /api/v1/events/{id}/tickets/{ticketId}   删除门票 🔐")
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
