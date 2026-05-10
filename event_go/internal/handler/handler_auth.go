package handler

import (
	"context"
	"encoding/json"
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "请求体格式错误"})
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
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: "密码加密失败"})
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
			writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		}
		return
	}

	token, err := h.generateToken(u)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: "令牌生成失败"})
		return
	}

	writeJSON(w, http.StatusCreated, model.APIResp{
		Code: 201, Message: "注册成功",
		Data: model.LoginResp{Token: token, User: *u},
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}
	if req.Contact == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "联系方式、密码不能为空"})
		return
	}

	u, err := h.store.GetUserByContact(req.Contact)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
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
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: "令牌生成失败"})
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
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			next.ServeHTTP(w, r)
			return
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		claims := &model.UserClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			next.ServeHTTP(w, r)
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
