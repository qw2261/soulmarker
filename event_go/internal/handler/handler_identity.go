package handler

import (
	"net/http"
	"strconv"

	"github.com/qw2261/soulmarker/event_go/internal/api"
	"github.com/qw2261/soulmarker/event_go/internal/handler/dto"
)

func (h *Handler) GetIdentityMigrationReport(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if value := r.URL.Query().Get("legacy_limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 || parsed > 100 {
			writeError(w, http.StatusBadRequest, api.CodeValidationError, "legacy_limit 必须在 1 到 100 之间")
			return
		}
		limit = parsed
	}

	report, err := h.store.GetIdentityMigrationReport(limit)
	if err != nil {
		writeInternalError(w, "get_identity_migration_report", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.Response{Code: 200, Message: "ok", Data: dto.IdentityMigrationReport(report)})
}
