package api

import "sort"

// ErrorCode 是与 HTTP 状态码独立、供客户端稳定判断失败原因的业务错误码。
type ErrorCode string

const (
	CodeValidationError              ErrorCode = "VALIDATION_ERROR"
	CodeInvalidJSON                  ErrorCode = "INVALID_JSON"
	CodeRequestTooLarge              ErrorCode = "REQUEST_TOO_LARGE"
	CodeAPIRouteNotFound             ErrorCode = "API_ROUTE_NOT_FOUND"
	CodeMethodNotAllowed             ErrorCode = "METHOD_NOT_ALLOWED"
	CodeUserAuthRequired             ErrorCode = "USER_AUTH_REQUIRED"
	CodeUserTokenInvalid             ErrorCode = "USER_TOKEN_INVALID"
	CodeAdminAuthInvalid             ErrorCode = "ADMIN_AUTH_INVALID"
	CodeInvalidCredentials           ErrorCode = "INVALID_CREDENTIALS"
	CodeUserAlreadyExists            ErrorCode = "USER_ALREADY_EXISTS"
	CodeEventNotFound                ErrorCode = "EVENT_NOT_FOUND"
	CodeOrganizerNotFound            ErrorCode = "ORGANIZER_NOT_FOUND"
	CodeTicketNotFound               ErrorCode = "TICKET_NOT_FOUND"
	CodePostNotFound                 ErrorCode = "POST_NOT_FOUND"
	CodeEventNotPublished            ErrorCode = "EVENT_NOT_PUBLISHED"
	CodeRegistrationDuplicate        ErrorCode = "REGISTRATION_DUPLICATE"
	CodeEventCapacityFull            ErrorCode = "EVENT_CAPACITY_FULL"
	CodeTicketSoldOut                ErrorCode = "TICKET_SOLD_OUT"
	CodeRegistrationNotFound         ErrorCode = "REGISTRATION_NOT_FOUND"
	CodeCancellationDeadlineExceeded ErrorCode = "CANCELLATION_DEADLINE_EXCEEDED"
	CodeParticipationRequired        ErrorCode = "PARTICIPATION_REQUIRED"
	CodeInternalError                ErrorCode = "INTERNAL_ERROR"
)

var errorMessages = map[ErrorCode]string{
	CodeValidationError:              "请求参数校验失败",
	CodeInvalidJSON:                  "请求体格式错误",
	CodeRequestTooLarge:              "请求体过大",
	CodeAPIRouteNotFound:             "API 路由不存在",
	CodeMethodNotAllowed:             "请求方法不允许",
	CodeUserAuthRequired:             "请先登录",
	CodeUserTokenInvalid:             "用户认证失败，请重新登录",
	CodeAdminAuthInvalid:             "认证失败，请提供有效的管理员令牌",
	CodeInvalidCredentials:           "联系方式或密码错误",
	CodeUserAlreadyExists:            "该联系方式已注册",
	CodeEventNotFound:                "活动不存在",
	CodeOrganizerNotFound:            "门店不存在",
	CodeTicketNotFound:               "门票不存在",
	CodePostNotFound:                 "帖子不存在",
	CodeEventNotPublished:            "活动未发布，暂无法报名",
	CodeRegistrationDuplicate:        "该联系方式已报名本活动",
	CodeEventCapacityFull:            "活动报名已满",
	CodeTicketSoldOut:                "门票已售罄",
	CodeRegistrationNotFound:         "未找到报名记录",
	CodeCancellationDeadlineExceeded: "已过取消截止时间，无法取消报名",
	CodeParticipationRequired:        "只有报名者才能参与讨论",
	CodeInternalError:                "服务器内部错误",
}

// ErrorCodes 返回错误码目录的副本，供契约生成和测试使用。
func ErrorCodes() []ErrorCode {
	codes := make([]ErrorCode, 0, len(errorMessages))
	for code := range errorMessages {
		codes = append(codes, code)
	}
	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })
	return codes
}

// DefaultErrorMessage 返回错误码对应的稳定默认消息。
func DefaultErrorMessage(code ErrorCode) (string, bool) {
	message, ok := errorMessages[code]
	return message, ok
}

// NewErrorResponse 构造统一错误响应。未知错误码会被安全地降级为通用内部错误。
func NewErrorResponse(numericCode int, code ErrorCode, message string) Response {
	defaultMessage, ok := DefaultErrorMessage(code)
	if !ok {
		code = CodeInternalError
		defaultMessage = errorMessages[CodeInternalError]
		message = defaultMessage
	}
	if message == "" {
		message = defaultMessage
	}
	return Response{
		Code:      numericCode,
		ErrorCode: string(code),
		Message:   message,
	}
}
