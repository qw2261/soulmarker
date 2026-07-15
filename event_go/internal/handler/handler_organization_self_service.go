package handler

import (
	"errors"
	"net/http"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/authorization"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/model"
	"github.com/qw2261/soulmarker/event_go/internal/service"
)

func (h *Handler) requireOrganizationFeature(w http.ResponseWriter) bool {
	if !h.config.OrganizationAuthEnabled {
		writeError(w, http.StatusNotFound, api.CodeAPIRouteNotFound, "")
		return false
	}
	return true
}

func writeOrganizationSelfServiceError(w http.ResponseWriter, operation string, err error) {
	switch {
	case errors.Is(err, service.ErrOrganizationInputInvalid), errors.Is(err, service.ErrOrganizationRoleInvalid):
		writeError(w, http.StatusBadRequest, api.CodeValidationError, err.Error())
	case errors.Is(err, model.ErrOrganizationSlugInUse):
		writeError(w, http.StatusConflict, api.CodeOrganizationSlugInUse, "")
	case errors.Is(err, model.ErrOrganizationInvitationExists):
		writeError(w, http.StatusConflict, api.CodeOrganizationInvitationExists, "")
	case errors.Is(err, model.ErrOrganizationInvitationInvalid):
		writeError(w, http.StatusBadRequest, api.CodeOrganizationInvitationInvalid, "")
	case errors.Is(err, model.ErrOrganizationMemberExists):
		writeError(w, http.StatusConflict, api.CodeOrganizationMemberExists, "")
	case errors.Is(err, model.ErrOrganizationMemberNotFound):
		writeError(w, http.StatusNotFound, api.CodeOrganizationMemberNotFound, "")
	case errors.Is(err, model.ErrOrganizationMemberChangeDenied), errors.Is(err, model.ErrOrganizationPermissionDenied):
		writeError(w, http.StatusForbidden, api.CodeOrganizationMemberChangeDenied, "")
	default:
		writeInternalError(w, operation, err)
	}
}

func (h *Handler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	if !h.requireOrganizationFeature(w) {
		return
	}
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}
	var request dto.CreateOrganizationRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	organization := &model.Organization{Name: request.Name, Slug: request.Slug}
	profile := &model.OrganizerProfile{
		Name: request.ProfileName, Description: request.ProfileDescription, Contact: request.ProfileContact,
		LogoURL: request.ProfileLogoURL, Address: request.ProfileAddress,
		Website: request.ProfileWebsite, Tags: request.ProfileTags,
	}
	if err := h.selfService.Create(user.ID, organization, profile); err != nil {
		writeOrganizationSelfServiceError(w, "create_organization", err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.Response{
		Code: http.StatusCreated, Message: "组织创建成功", Data: dto.OrganizationWorkspace(organization, profile),
	})
}

func (h *Handler) GetOrganizationWorkspace(w http.ResponseWriter, r *http.Request) {
	value, ok := authorization.OrganizationContextFromContext(r.Context())
	if !ok {
		writeInternalError(w, "organization_context_missing", errors.New("organization context missing"))
		return
	}
	organization, profile, err := h.selfService.Get(value.OrganizationID)
	if err != nil {
		writeInternalError(w, "get_organization_workspace", err)
		return
	}
	if organization == nil || profile == nil {
		writeError(w, http.StatusNotFound, api.CodeAPIRouteNotFound, "")
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{
		Code: http.StatusOK, Message: "ok", Data: dto.OrganizationWorkspace(organization, profile),
	})
}

func (h *Handler) ListOrganizationMembers(w http.ResponseWriter, r *http.Request) {
	value, ok := authorization.OrganizationContextFromContext(r.Context())
	if !ok {
		writeInternalError(w, "organization_context_missing", errors.New("organization context missing"))
		return
	}
	members, err := h.selfService.ListMembers(value.OrganizationID)
	if err != nil {
		writeInternalError(w, "list_organization_members", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: http.StatusOK, Message: "ok", Data: dto.OrganizationMembers(members)})
}

func (h *Handler) UpdateOrganizationMember(w http.ResponseWriter, r *http.Request) {
	value, ok := authorization.OrganizationContextFromContext(r.Context())
	if !ok {
		writeInternalError(w, "organization_context_missing", errors.New("organization context missing"))
		return
	}
	memberID, err := parseOrganizationMemberID(r)
	if err != nil || memberID <= 0 {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的成员 ID")
		return
	}
	var request dto.UpdateOrganizationMemberRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := h.selfService.UpdateMemberRole(value.OrganizationID, value.UserID, memberID, request.Role); err != nil {
		writeOrganizationSelfServiceError(w, "update_organization_member", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: http.StatusOK, Message: "成员角色已更新"})
}

func (h *Handler) RevokeOrganizationMember(w http.ResponseWriter, r *http.Request) {
	value, ok := authorization.OrganizationContextFromContext(r.Context())
	if !ok {
		writeInternalError(w, "organization_context_missing", errors.New("organization context missing"))
		return
	}
	memberID, err := parseOrganizationMemberID(r)
	if err != nil || memberID <= 0 {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的成员 ID")
		return
	}
	if err := h.selfService.RevokeMember(value.OrganizationID, value.UserID, memberID); err != nil {
		writeOrganizationSelfServiceError(w, "revoke_organization_member", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: http.StatusOK, Message: "成员已撤销"})
}

func (h *Handler) CreateOrganizationInvitation(w http.ResponseWriter, r *http.Request) {
	value, ok := authorization.OrganizationContextFromContext(r.Context())
	if !ok {
		writeInternalError(w, "organization_context_missing", errors.New("organization context missing"))
		return
	}
	var request dto.CreateOrganizationInvitationRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	invitation, err := h.selfService.Invite(
		r.Context(), value.OrganizationID, value.UserID, request.Email, request.Role,
	)
	if err != nil {
		writeOrganizationSelfServiceError(w, "create_organization_invitation", err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.Response{
		Code: http.StatusCreated, Message: "邀请已发送", Data: dto.OrganizationInvitation(invitation),
	})
}

func (h *Handler) ListOrganizationInvitations(w http.ResponseWriter, r *http.Request) {
	value, ok := authorization.OrganizationContextFromContext(r.Context())
	if !ok {
		writeInternalError(w, "organization_context_missing", errors.New("organization context missing"))
		return
	}
	invitations, err := h.selfService.ListInvitations(value.OrganizationID)
	if err != nil {
		writeInternalError(w, "list_organization_invitations", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{
		Code: http.StatusOK, Message: "ok", Data: dto.OrganizationInvitations(invitations),
	})
}

func (h *Handler) RevokeOrganizationInvitation(w http.ResponseWriter, r *http.Request) {
	value, ok := authorization.OrganizationContextFromContext(r.Context())
	if !ok {
		writeInternalError(w, "organization_context_missing", errors.New("organization context missing"))
		return
	}
	invitationID, err := parseOrganizationInvitationID(r)
	if err != nil || invitationID <= 0 {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的邀请 ID")
		return
	}
	if err := h.selfService.RevokeInvitation(value.OrganizationID, value.UserID, invitationID); err != nil {
		writeOrganizationSelfServiceError(w, "revoke_organization_invitation", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: http.StatusOK, Message: "邀请已撤销"})
}

func (h *Handler) AcceptOrganizationInvitation(w http.ResponseWriter, r *http.Request) {
	if !h.requireOrganizationFeature(w) {
		return
	}
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}
	var request dto.AcceptOrganizationInvitationRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := h.selfService.Accept(request.Token, user.ID); err != nil {
		writeOrganizationSelfServiceError(w, "accept_organization_invitation", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: http.StatusOK, Message: "已加入组织"})
}
