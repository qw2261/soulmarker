package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/auth"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/model"
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
	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "密码至少 6 位")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeInternalError(w, "register_hash_password", err)
		return
	}

	u := &model.User{
		Name:         req.Name,
		Contact:      req.Contact,
		PasswordHash: string(hash),
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

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Contact == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "联系方式、密码不能为空")
		return
	}

	u, err := h.store.GetUserByContact(req.Contact)
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
	return user, true
}
