package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/qw2261/soulmarker/event_go/internal/config"
	"github.com/qw2261/soulmarker/event_go/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterUserReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Name == "" || req.Contact == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "姓名、联系方式、密码不能为空"})
		return
	}
	if len(req.Password) < 6 {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "密码至少 6 位"})
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
			writeJSON(w, http.StatusConflict, model.APIResp{Code: 409, Message: err.Error()})
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

	writeJSON(w, http.StatusCreated, model.APIResp{
		Code: 201, Message: "注册成功",
		Data: model.LoginResp{Token: token, User: *u},
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Contact == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "联系方式、密码不能为空"})
		return
	}

	u, err := h.store.GetUserByContact(req.Contact)
	if err != nil {
		writeInternalError(w, "login_get_user", err)
		return
	}
	if u == nil {
		writeJSON(w, http.StatusUnauthorized, model.APIResp{Code: 401, Message: model.ErrInvalidCreds.Error()})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, model.APIResp{Code: 401, Message: model.ErrInvalidCreds.Error()})
		return
	}

	token, err := h.generateToken(u)
	if err != nil {
		writeInternalError(w, "login_generate_token", err)
		return
	}

	writeJSON(w, http.StatusOK, model.APIResp{
		Code: 200, Message: "登录成功",
		Data: model.LoginResp{Token: token, User: *u},
	})
}

func (h *Handler) generateToken(u *model.User) (string, error) {
	cfg := config.Load()
	claims := &model.UserClaims{
		UserID:  u.ID,
		Name:    u.Name,
		Contact: u.Contact,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.JWTExpireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

func UserAuth(next http.Handler) http.Handler {
	cfg := config.Load()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			next.ServeHTTP(w, r)
			return
		}
		if !strings.HasPrefix(auth, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, model.APIResp{Code: 401, Message: "用户认证失败，请重新登录"})
			return
		}

		tokenStr := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if tokenStr == "" {
			writeJSON(w, http.StatusUnauthorized, model.APIResp{Code: 401, Message: "用户认证失败，请重新登录"})
			return
		}
		claims := &model.UserClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || !token.Valid {
			writeJSON(w, http.StatusUnauthorized, model.APIResp{Code: 401, Message: "用户认证失败，请重新登录"})
			return
		}

		ctx := context.WithValue(r.Context(), model.UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getUserIdentity(r *http.Request) (name, contact string) {
	if claims, ok := model.UserFromContext(r.Context()); ok {
		return claims.Name, claims.Contact
	}
	return "", ""
}
