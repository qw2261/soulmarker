package handler

import (
	"net/http"

	"github.com/qw2261/soulmarker/event_go/internal/authorization"
)

func fullPIIAccess(r *http.Request) bool {
	if authorization.IsPlatformAdmin(r.Context()) {
		return true
	}
	value, ok := authorization.OrganizationContextFromContext(r.Context())
	if !ok {
		return false
	}
	for _, capability := range value.Capabilities {
		if capability == authorization.CapabilityFinanceRead {
			return true
		}
	}
	return false
}
