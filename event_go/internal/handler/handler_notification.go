package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}
	unreadOnly := false
	if raw := r.URL.Query().Get("unread_only"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, api.CodeValidationError, "unread_only 必须是 true 或 false")
			return
		}
		unreadOnly = parsed
	}
	page, pageSize := parsePagination(r)
	notifications, total, err := h.notifications.List(user.ID, unreadOnly, (page-1)*pageSize, pageSize)
	if err != nil {
		writeInternalError(w, "list_notifications", err)
		return
	}
	paginatedOK(w, dto.Notifications(notifications), total, page, pageSize)
}

func (h *Handler) GetNotificationUnreadCount(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}
	unread, err := h.notifications.UnreadCount(user.ID)
	if err != nil {
		writeInternalError(w, "count_unread_notifications", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{
		Code: http.StatusOK, Message: "ok", Data: dto.NotificationUnreadCountResponse{Unread: unread},
	})
}

func (h *Handler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}
	notificationID, err := parseNotificationID(r)
	if err != nil || notificationID <= 0 {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的通知 ID")
		return
	}
	if err := h.notifications.MarkRead(user.ID, notificationID); err != nil {
		if errors.Is(err, model.ErrNotificationNotFound) {
			writeError(w, http.StatusNotFound, api.CodeNotificationNotFound, "")
		} else {
			writeInternalError(w, "mark_notification_read", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: http.StatusOK, Message: "通知已读"})
}

func (h *Handler) MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}
	updated, err := h.notifications.MarkAllRead(user.ID)
	if err != nil {
		writeInternalError(w, "mark_all_notifications_read", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{
		Code: http.StatusOK, Message: "全部通知已读",
		Data: dto.NotificationsMarkedReadResponse{Updated: updated},
	})
}
