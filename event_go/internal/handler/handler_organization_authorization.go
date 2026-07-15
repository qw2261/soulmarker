package handler

import (
	"errors"
	"net/http"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/authorization"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
)

func (h *Handler) OrganizationAuth(next http.Handler, capability authorization.Capability) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.config.OrganizationAuthEnabled {
			writeError(w, http.StatusNotFound, api.CodeAPIRouteNotFound, "")
			return
		}
		user, authenticated := h.requireUser(w, r)
		if !authenticated {
			return
		}
		organizationID, err := parseOrganizationID(r)
		if err != nil || organizationID <= 0 {
			writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的组织 ID")
			return
		}
		value, err := h.organizations.Authorize(user.ID, organizationID, capability)
		if err != nil {
			if errors.Is(err, authorization.ErrOrganizationAccessDenied) {
				writeError(w, http.StatusForbidden, api.CodeOrganizationAccessDenied, "")
			} else {
				writeInternalError(w, "authorize_organization", err)
			}
			return
		}
		ctx := authorization.WithOrganizationContext(r.Context(), *value)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) ListMyOrganizations(w http.ResponseWriter, r *http.Request) {
	if !h.config.OrganizationAuthEnabled {
		writeError(w, http.StatusNotFound, api.CodeAPIRouteNotFound, "")
		return
	}
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}
	contexts, err := h.organizations.ListForUser(user.ID)
	if err != nil {
		writeInternalError(w, "list_my_organizations", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{
		Code: http.StatusOK, Message: "ok", Data: dto.OrganizationContexts(contexts),
	})
}

func (h *Handler) GetOrganizationSession(w http.ResponseWriter, r *http.Request) {
	value, ok := authorization.OrganizationContextFromContext(r.Context())
	if !ok {
		writeInternalError(w, "organization_context_missing", errors.New("organization context missing"))
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{
		Code: http.StatusOK, Message: "组织身份有效", Data: dto.OrganizationContext(value),
	})
}
