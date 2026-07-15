package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/auth"
	"github.com/qw2261/soulmarker/event_go/internal/emailaddr"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/model"
	"github.com/qw2261/soulmarker/event_go/internal/service"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterUserRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Name == "" || req.Contact == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "姓名、联系方式、密码不能为空")
		return
	}
	if !service.ValidPassword(req.Password) {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "密码必须为 8 到 72 个字节")
		return
	}
	normalizedEmail, valid := emailaddr.Normalize(req.Contact)
	if !valid {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "请使用有效邮箱注册")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeInternalError(w, "register_hash_password", err)
		return
	}

	u := &model.User{
		Name:          req.Name,
		Contact:       normalizedEmail,
		RecoveryEmail: normalizedEmail,
		PasswordHash:  string(hash),
	}
	if err := h.store.CreateUser(u); err != nil {
		if errors.Is(err, model.ErrUserExists) {
			writeError(w, http.StatusConflict, api.CodeUserAlreadyExists, err.Error())
		} else {
			writeInternalError(w, "register_user", err)
		}
		return
	}

	token, err := h.generateToken(u)
	if err != nil {
		writeInternalError(w, "register_generate_token", err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.Response{
		Code: 201, Message: "注册成功",
		Data: dto.LoginResponse{Token: token, User: dto.User(u)},
	})
}

func (h *Handler) RequestRecoveryEmail(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}
	var req dto.RecoveryEmailRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Email) == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "恢复邮箱和当前密码不能为空")
		return
	}
	if err := h.authentication.RequestRecoveryEmail(r.Context(), user, req.Email, req.Password); err != nil {
		switch {
		case errors.Is(err, service.ErrRecoveryEmailFormatInvalid):
			writeError(w, http.StatusBadRequest, api.CodeValidationError, err.Error())
		case errors.Is(err, model.ErrInvalidCreds):
			writeError(w, http.StatusUnauthorized, api.CodeInvalidCredentials, "当前密码错误")
		case errors.Is(err, model.ErrRecoveryEmailInUse):
			writeError(w, http.StatusConflict, api.CodeRecoveryEmailInUse, "")
		case errors.Is(err, model.ErrRecoveryEmailBound):
			writeError(w, http.StatusConflict, api.CodeRecoveryEmailAlreadyBound, "")
		case errors.Is(err, model.ErrRecoveryEmailRateLimit):
			writeError(w, http.StatusTooManyRequests, api.CodeRecoveryEmailRateLimited, "")
		default:
			writeInternalError(w, "request_recovery_email", err)
		}
		return
	}
	writeJSON(w, http.StatusAccepted, dto.Response{
		Code: http.StatusAccepted, Message: "验证邮件已发送，请在有效期内完成绑定",
	})
}

func (h *Handler) ConfirmRecoveryEmail(w http.ResponseWriter, r *http.Request) {
	var req dto.RecoveryEmailConfirmRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.authentication.ConfirmRecoveryEmail(req.Token); err != nil {
		switch {
		case errors.Is(err, model.ErrRecoveryEmailInvalid):
			writeError(w, http.StatusBadRequest, api.CodeRecoveryEmailTokenInvalid, "")
		case errors.Is(err, model.ErrRecoveryEmailInUse):
			writeError(w, http.StatusConflict, api.CodeRecoveryEmailInUse, "")
		default:
			writeInternalError(w, "confirm_recovery_email", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{
		Code: http.StatusOK, Message: "恢复邮箱已绑定，旧会话已失效，请重新登录",
	})
}

func (h *Handler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req dto.PasswordResetRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Contact) == "" {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "邮箱不能为空")
		return
	}
	if err := h.authentication.RequestPasswordReset(r.Context(), strings.ToLower(strings.TrimSpace(req.Contact))); err != nil {
		if errors.Is(err, model.ErrPasswordResetRateLimit) {
			slog.Warn("password reset request rate limited")
		} else {
			slog.Error("password reset request failed", "error", err)
		}
	}
	writeJSON(w, http.StatusAccepted, dto.Response{
		Code: http.StatusAccepted, Message: "如果该邮箱已注册，重置链接将发送到对应邮箱",
	})
}

func (h *Handler) ConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req dto.PasswordResetConfirmRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !service.ValidPassword(req.Password) {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "密码必须为 8 到 72 个字节")
		return
	}
	if err := h.authentication.ResetPassword(req.Token, req.Password); err != nil {
		if errors.Is(err, model.ErrPasswordResetInvalid) {
			writeError(w, http.StatusBadRequest, api.CodePasswordResetInvalid, "")
		} else {
			writeInternalError(w, "confirm_password_reset", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: http.StatusOK, Message: "密码已重置，请重新登录"})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}
	if err := h.authentication.Logout(user.ID); err != nil {
		writeInternalError(w, "logout_user", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: http.StatusOK, Message: "已退出登录"})
}

func (h *Handler) GetAdminSession(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, dto.Response{
		Code: http.StatusOK, Message: "管理员身份有效",
		Data: dto.AdminSessionResponse{Authenticated: true},
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Contact == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "联系方式、密码不能为空")
		return
	}

	u, err := h.store.GetUserByContact(strings.ToLower(strings.TrimSpace(req.Contact)))
	if err != nil {
		writeInternalError(w, "login_get_user", err)
		return
	}
	if u == nil {
		writeError(w, http.StatusUnauthorized, api.CodeInvalidCredentials, model.ErrInvalidCreds.Error())
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, api.CodeInvalidCredentials, model.ErrInvalidCreds.Error())
		return
	}

	token, err := h.generateToken(u)
	if err != nil {
		writeInternalError(w, "login_generate_token", err)
		return
	}

	writeJSON(w, http.StatusOK, dto.Response{
		Code: 200, Message: "登录成功",
		Data: dto.LoginResponse{Token: token, User: dto.User(u)},
	})
}

func (h *Handler) generateToken(u *model.User) (string, error) {
	return h.tokens.SignUser(u, h.clock.Now(), time.Duration(h.config.JWTExpireHours)*time.Hour)
}

func UserAuth(next http.Handler, tokens auth.TokenManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			next.ServeHTTP(w, r)
			return
		}
		if !strings.HasPrefix(auth, "Bearer ") {
			writeError(w, http.StatusUnauthorized, api.CodeUserTokenInvalid, "")
			return
		}

		tokenStr := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, api.CodeUserTokenInvalid, "")
			return
		}
		claims, err := tokens.VerifyUser(tokenStr)
		if err != nil {
			writeError(w, http.StatusUnauthorized, api.CodeUserTokenInvalid, "")
			return
		}

		ctx := context.WithValue(r.Context(), model.UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) requireUser(w http.ResponseWriter, r *http.Request) (*model.User, bool) {
	claims, ok := model.UserFromContext(r.Context())
	if !ok || claims.UserID <= 0 {
		writeError(w, http.StatusUnauthorized, api.CodeUserAuthRequired, "")
		return nil, false
	}
	user, err := h.store.GetUserByID(claims.UserID)
	if err != nil {
		writeInternalError(w, "load_authenticated_user", err)
		return nil, false
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, api.CodeUserTokenInvalid, "用户不存在或登录已失效")
		return nil, false
	}
	claimVersion := claims.AuthVersion
	if claimVersion == 0 {
		claimVersion = 1
	}
	if claimVersion != user.AuthVersion {
		writeError(w, http.StatusUnauthorized, api.CodeUserTokenInvalid, "登录已失效，请重新登录")
		return nil, false
	}
	return user, true
}
