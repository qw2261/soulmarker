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
	CodeReplyNotFound                ErrorCode = "REPLY_NOT_FOUND"
	CodeEventNotPublished            ErrorCode = "EVENT_NOT_PUBLISHED"
	CodeRegistrationDuplicate        ErrorCode = "REGISTRATION_DUPLICATE"
	CodeEventCapacityFull            ErrorCode = "EVENT_CAPACITY_FULL"
	CodeTicketSoldOut                ErrorCode = "TICKET_SOLD_OUT"
	CodeRegistrationNotFound         ErrorCode = "REGISTRATION_NOT_FOUND"
	CodeCancellationDeadlineExceeded ErrorCode = "CANCELLATION_DEADLINE_EXCEEDED"
	CodeParticipationRequired        ErrorCode = "PARTICIPATION_REQUIRED"
	CodeInternalError                ErrorCode = "INTERNAL_ERROR"
	CodeAdmissionNotFound            ErrorCode = "ADMISSION_NOT_FOUND"
	CodeAdmissionRevoked             ErrorCode = "ADMISSION_REVOKED"
	CodeAdmissionAlreadyCheckedIn    ErrorCode = "ADMISSION_ALREADY_CHECKED_IN"
	CodeEventHasAdmissions           ErrorCode = "EVENT_HAS_ADMISSIONS"
	CodePasswordResetInvalid         ErrorCode = "PASSWORD_RESET_TOKEN_INVALID"
	CodeRecoveryEmailInUse           ErrorCode = "RECOVERY_EMAIL_IN_USE"
	CodeRecoveryEmailAlreadyBound    ErrorCode = "RECOVERY_EMAIL_ALREADY_BOUND"
	CodeRecoveryEmailTokenInvalid    ErrorCode = "RECOVERY_EMAIL_TOKEN_INVALID"
	CodeRecoveryEmailRateLimited     ErrorCode = "RECOVERY_EMAIL_RATE_LIMITED"
	CodeNotificationNotFound         ErrorCode = "NOTIFICATION_NOT_FOUND"
	CodeContentReportNotFound        ErrorCode = "CONTENT_REPORT_NOT_FOUND"
	CodeContentReportNotAllowed      ErrorCode = "CONTENT_REPORT_NOT_ALLOWED"
	CodeContentReportAlreadyResolved ErrorCode = "CONTENT_REPORT_ALREADY_RESOLVED"
	CodeContentAlreadyRemoved        ErrorCode = "CONTENT_ALREADY_REMOVED"
	CodeContentAlreadyVisible        ErrorCode = "CONTENT_ALREADY_VISIBLE"
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
	CodeReplyNotFound:                "回复不存在",
	CodeEventNotPublished:            "活动未发布，暂无法报名",
	CodeRegistrationDuplicate:        "该联系方式已报名本活动",
	CodeEventCapacityFull:            "活动报名已满",
	CodeTicketSoldOut:                "门票已售罄",
	CodeRegistrationNotFound:         "未找到报名记录",
	CodeCancellationDeadlineExceeded: "已过取消截止时间，无法取消报名",
	CodeParticipationRequired:        "只有报名者才能参与讨论",
	CodeInternalError:                "服务器内部错误",
	CodeAdmissionNotFound:            "入场凭证不存在",
	CodeAdmissionRevoked:             "入场凭证已失效",
	CodeAdmissionAlreadyCheckedIn:    "入场凭证已核销",
	CodeEventHasAdmissions:           "活动已有入场凭证，不能删除",
	CodePasswordResetInvalid:         "密码重置链接无效或已过期",
	CodeRecoveryEmailInUse:           "该邮箱已绑定其他账户",
	CodeRecoveryEmailAlreadyBound:    "该邮箱已绑定当前账户",
	CodeRecoveryEmailTokenInvalid:    "恢复邮箱验证链接无效或已过期",
	CodeRecoveryEmailRateLimited:     "恢复邮箱验证请求过于频繁",
	CodeNotificationNotFound:         "通知不存在",
	CodeContentReportNotFound:        "举报记录不存在",
	CodeContentReportNotAllowed:      "当前用户不能举报该内容",
	CodeContentReportAlreadyResolved: "举报记录已处理",
	CodeContentAlreadyRemoved:        "内容已被移除",
	CodeContentAlreadyVisible:        "内容已处于可见状态",
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
