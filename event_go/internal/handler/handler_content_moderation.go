package handler

import (
	"errors"
	"net/http"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/model"
	"github.com/qw2261/soulmarker/event_go/internal/service"
)

const platformAdminActor = "platform_admin"

func writeContentReportError(w http.ResponseWriter, operation string, err error, targetType string) {
	switch {
	case errors.Is(err, model.ErrContentTargetNotFound):
		if targetType == model.ContentTargetReply {
			writeError(w, http.StatusNotFound, api.CodeReplyNotFound, "")
		} else {
			writeError(w, http.StatusNotFound, api.CodePostNotFound, "")
		}
	case errors.Is(err, model.ErrNotRegistered):
		writeError(w, http.StatusForbidden, api.CodeParticipationRequired, err.Error())
	case errors.Is(err, service.ErrContentReportOwnContent):
		writeError(w, http.StatusForbidden, api.CodeContentReportNotAllowed, err.Error())
	case errors.Is(err, service.ErrContentReportCategoryInvalid),
		errors.Is(err, service.ErrContentReportDetailRequired),
		errors.Is(err, service.ErrContentReportDetailTooLong),
		errors.Is(err, service.ErrContentModerationNoteRequired),
		errors.Is(err, service.ErrContentModerationNoteTooLong),
		errors.Is(err, service.ErrContentResolutionInvalid),
		errors.Is(err, service.ErrContentReportStatusInvalid),
		errors.Is(err, service.ErrContentTargetTypeInvalid):
		writeError(w, http.StatusBadRequest, api.CodeValidationError, err.Error())
	case errors.Is(err, model.ErrContentReportNotFound):
		writeError(w, http.StatusNotFound, api.CodeContentReportNotFound, "")
	case errors.Is(err, model.ErrContentReportResolved):
		writeError(w, http.StatusConflict, api.CodeContentReportAlreadyResolved, "")
	case errors.Is(err, model.ErrContentAlreadyRemoved):
		writeError(w, http.StatusConflict, api.CodeContentAlreadyRemoved, "")
	case errors.Is(err, model.ErrContentAlreadyVisible):
		writeError(w, http.StatusConflict, api.CodeContentAlreadyVisible, "")
	default:
		writeInternalError(w, operation, err)
	}
}

func (h *Handler) ReportPost(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}
	eventID, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}
	postID, err := parsePostID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的帖子 ID")
		return
	}
	var req dto.CreateContentReportRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	report, created, err := h.moderation.ReportPost(eventID, postID, user, req.Category, req.Detail)
	if err != nil {
		writeContentReportError(w, "report_post", err, model.ContentTargetPost)
		return
	}
	status := http.StatusCreated
	message := "举报已提交"
	if !created {
		status = http.StatusOK
		message = "该内容已有未处理举报"
	}
	writeJSON(w, status, dto.Response{Code: status, Message: message, Data: dto.ContentReportReceipt(report)})
}

func (h *Handler) ReportReply(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}
	eventID, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}
	postID, err := parsePostID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的帖子 ID")
		return
	}
	replyID, err := parseReplyID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的回复 ID")
		return
	}
	var req dto.CreateContentReportRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	report, created, err := h.moderation.ReportReply(eventID, postID, replyID, user, req.Category, req.Detail)
	if err != nil {
		writeContentReportError(w, "report_reply", err, model.ContentTargetReply)
		return
	}
	status := http.StatusCreated
	message := "举报已提交"
	if !created {
		status = http.StatusOK
		message = "该内容已有未处理举报"
	}
	writeJSON(w, status, dto.Response{Code: status, Message: message, Data: dto.ContentReportReceipt(report)})
}

func (h *Handler) ListContentReports(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePagination(r)
	status := r.URL.Query().Get("status")
	if status == "" {
		status = model.ContentReportStatusOpen
	}
	reports, total, err := h.moderation.ListReports(model.ListContentReportsParams{
		Status: status, TargetType: r.URL.Query().Get("target_type"),
		Offset: (page - 1) * pageSize, Limit: pageSize,
	})
	if err != nil {
		writeContentReportError(w, "list_content_reports", err, "")
		return
	}
	paginatedOK(w, dto.ContentReports(reports), total, page, pageSize)
}

func (h *Handler) ResolveContentReport(w http.ResponseWriter, r *http.Request) {
	reportID, err := parseReportID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的举报 ID")
		return
	}
	var req dto.ResolveContentReportRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	report, err := h.moderation.ResolveReport(reportID, req.Resolution, req.Note, platformAdminActor)
	if err != nil {
		writeContentReportError(w, "resolve_content_report", err, "")
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: http.StatusOK, Message: "举报已处理", Data: dto.ContentReport(report)})
}

func (h *Handler) moderatePost(w http.ResponseWriter, r *http.Request, remove bool) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}
	postID, err := parsePostID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的帖子 ID")
		return
	}
	var req dto.ModerateContentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if remove {
		err = h.moderation.RemovePost(eventID, postID, platformAdminActor, req.Reason)
	} else {
		err = h.moderation.RestorePost(eventID, postID, platformAdminActor, req.Reason)
	}
	if err != nil {
		writeContentReportError(w, "moderate_post", err, model.ContentTargetPost)
		return
	}
	message := "帖子已恢复"
	if remove {
		message = "帖子已移除"
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: http.StatusOK, Message: message})
}

func (h *Handler) RemovePost(w http.ResponseWriter, r *http.Request) {
	h.moderatePost(w, r, true)
}

func (h *Handler) RestorePost(w http.ResponseWriter, r *http.Request) {
	h.moderatePost(w, r, false)
}

func (h *Handler) moderateReply(w http.ResponseWriter, r *http.Request, remove bool) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的活动 ID")
		return
	}
	postID, err := parsePostID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的帖子 ID")
		return
	}
	replyID, err := parseReplyID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, api.CodeValidationError, "无效的回复 ID")
		return
	}
	var req dto.ModerateContentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if remove {
		err = h.moderation.RemoveReply(eventID, postID, replyID, platformAdminActor, req.Reason)
	} else {
		err = h.moderation.RestoreReply(eventID, postID, replyID, platformAdminActor, req.Reason)
	}
	if err != nil {
		writeContentReportError(w, "moderate_reply", err, model.ContentTargetReply)
		return
	}
	message := "回复已恢复"
	if remove {
		message = "回复已移除"
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: http.StatusOK, Message: message})
}

func (h *Handler) RemoveReply(w http.ResponseWriter, r *http.Request) {
	h.moderateReply(w, r, true)
}

func (h *Handler) RestoreReply(w http.ResponseWriter, r *http.Request) {
	h.moderateReply(w, r, false)
}

func (h *Handler) ListContentModerationActions(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePagination(r)
	actions, total, err := h.moderation.ListActions((page-1)*pageSize, pageSize)
	if err != nil {
		writeInternalError(w, "list_content_moderation_actions", err)
		return
	}
	paginatedOK(w, dto.ContentModerationActions(actions), total, page, pageSize)
}
