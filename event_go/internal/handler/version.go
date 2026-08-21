package handler

import (
	"net/http"

	"github.com/qw2261/soulmarker/event_go/internal/buildinfo"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
)

// VersionHandler 返回构建 provenance（版本、Commit SHA、构建时间、Go 版本、依赖），
// 用于让发布制品、Tag、Commit、测试报告与部署记录之间能够互相追溯。
func (h *Handler) VersionHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, dto.Response{Code: 200, Message: "ok", Data: buildinfo.Result()})
}
