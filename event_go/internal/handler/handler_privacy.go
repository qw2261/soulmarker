package handler

import (
	"errors"
	"net/http"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

// ListMyPrivacyRequests 列出当前用户发起的数据主体请求（账号注销/数据导出）及其处置状态。
func (h *Handler) ListMyPrivacyRequests(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}
	requests, err := h.store.ListDataSubjectRequests(user.ID)
	if err != nil {
		writeInternalError(w, "list_data_subject_requests", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{
		Code: http.StatusOK, Message: "ok", Data: dto.DataSubjectRequests(requests),
	})
}

// ExportMyData 记录一次数据导出请求，并返回该用户在系统中的全部个人数据负载（数据可携带权）。
func (h *Handler) ExportMyData(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}
	if _, err := h.store.CreateDataSubjectRequest(user.ID, model.DataSubjectRequestDataExport); err != nil {
		writeInternalError(w, "create_data_export_request", err)
		return
	}
	export, err := h.store.ExportUserData(user.ID)
	if err != nil {
		writeInternalError(w, "export_user_data", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{
		Code: http.StatusOK, Message: "数据导出成功", Data: export,
	})
}

// RequestAccountErasure 发起账号注销：创建 account_erasure 请求、标记 deleted_at 并撤销全部会话。
// 完成后当前登录态立即失效，后续请求将被拒绝。
func (h *Handler) RequestAccountErasure(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}
	request, err := h.store.ErasureUser(user.ID)
	if err != nil {
		if errors.Is(err, model.ErrUserAlreadyDeleted) {
			writeError(w, http.StatusConflict, api.CodeUserAlreadyDeleted, "")
			return
		}
		writeInternalError(w, "request_account_erasure", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{
		Code: http.StatusOK, Message: "账号注销已受理，全部会话已失效", Data: dto.DataSubjectRequest(request),
	})
}
