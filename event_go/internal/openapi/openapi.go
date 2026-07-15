package openapi

import (
	_ "embed"
	"net/http"
)

//go:embed v1.json
var V1 []byte

func ServeV1(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/vnd.oai.openapi+json;version=3.1")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(V1)
}
