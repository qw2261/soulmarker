package api

import "testing"

func TestErrorCodeCatalog(t *testing.T) {
	codes := ErrorCodes()
	if len(codes) != 46 {
		t.Fatalf("expected 46 stable error codes, got %d", len(codes))
	}
	seen := make(map[ErrorCode]bool, len(codes))
	for _, code := range codes {
		if seen[code] {
			t.Fatalf("duplicate error code %q", code)
		}
		seen[code] = true
		message, ok := DefaultErrorMessage(code)
		if !ok || message == "" {
			t.Fatalf("error code %q has no default message", code)
		}
	}
}

func TestNewErrorResponsePreservesLegacyNumericCode(t *testing.T) {
	response := NewErrorResponse(409, CodeRegistrationDuplicate, "")
	if response.Code != 409 {
		t.Fatalf("expected numeric code 409, got %d", response.Code)
	}
	if response.ErrorCode != string(CodeRegistrationDuplicate) {
		t.Fatalf("unexpected business error code %q", response.ErrorCode)
	}
	if response.Message == "" {
		t.Fatal("expected catalog default message")
	}
}

func TestNewErrorResponseRejectsUnknownCode(t *testing.T) {
	response := NewErrorResponse(500, ErrorCode("DATABASE_FAILURE"), "sql: secret")
	if response.ErrorCode != string(CodeInternalError) || response.Message != "服务器内部错误" {
		t.Fatalf("unknown code must degrade safely, got %+v", response)
	}
}
