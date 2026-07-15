package api

type Response struct {
	Code      int         `json:"code"`
	ErrorCode string      `json:"error_code,omitempty"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Total     *int        `json:"total,omitempty"`
	Page      *int        `json:"page,omitempty"`
	PageSize  *int        `json:"page_size,omitempty"`
}
