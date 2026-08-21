package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func auditResourceType(path string) string {
	switch {
	case strings.Contains(path, "/owner-transfer"):
		return "ownership"
	case strings.Contains(path, "/members"):
		return "member"
	case strings.Contains(path, "/invitations"):
		return "invitation"
	case strings.Contains(path, "/audits"):
		return "audit_log"
	case strings.Contains(path, "/tickets"):
		return "ticket"
	case strings.Contains(path, "/registrations/export"):
		return "registration_export"
	case strings.Contains(path, "/registrations"):
		return "registration"
	case strings.Contains(path, "/checkins"):
		return "checkin"
	case strings.Contains(path, "/events"):
		return "event"
	default:
		return "organization"
	}
}

func auditResourceID(r *http.Request, resourceType string) string {
	keys := []string{"memberId", "invitationId", "ticketId", "id", "organizationId"}
	if resourceType == "organization" || resourceType == "audit_log" {
		keys = []string{"organizationId"}
	}
	for _, key := range keys {
		if value := r.PathValue(key); value != "" {
			return value
		}
	}
	return ""
}

func auditOutcome(status int) string {
	if status >= 200 && status < 400 {
		return model.AuditOutcomeSuccess
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return model.AuditOutcomeDenied
	}
	return model.AuditOutcomeFailure
}

func (h *Handler) appendOrganizationAudit(r *http.Request, entry *model.OrganizationAuditLog) {
	if entry.RequestID == "" {
		entry.RequestID = RequestIDFromContext(r.Context())
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = h.clock.Now()
	}
	if err := h.store.AppendOrganizationAudit(entry); err != nil {
		slog.Error("append organization audit", "request_id", entry.RequestID, "action", entry.Action, "error", err)
	}
}

func (h *Handler) OrganizationAudit(action, resourceType string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, r)
		organizationID, err := strconv.ParseInt(r.PathValue("organizationId"), 10, 64)
		if err != nil || organizationID <= 0 {
			return
		}
		entry := &model.OrganizationAuditLog{
			OrganizationID: organizationID,
			ActorType:      model.AuditActorOrganizationMember,
			Action:         action,
			ResourceType:   resourceType,
			ResourceID:     auditResourceID(r, resourceType),
			RequestID:      RequestIDFromContext(r.Context()),
			Outcome:        auditOutcome(recorder.statusCode),
			HTTPStatus:     recorder.statusCode,
			CreatedAt:      h.clock.Now(),
		}
		if claims, ok := model.UserFromContext(r.Context()); ok && claims.UserID > 0 {
			entry.ActorID = &claims.UserID
		}
		h.appendOrganizationAudit(r, entry)
	})
}

func (h *Handler) ListOrganizationAudits(w http.ResponseWriter, r *http.Request) {
	organizationID, err := parseOrganizationID(r)
	if err != nil || organizationID <= 0 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "无效的组织 ID")
		return
	}
	page, pageSize := parsePagination(r)
	entries, total, err := h.store.ListOrganizationAudits(organizationID, (page-1)*pageSize, pageSize)
	if err != nil {
		writeInternalError(w, "list_organization_audits", err)
		return
	}
	paginatedOK(w, dto.OrganizationAudits(entries), total, page, pageSize)
}
