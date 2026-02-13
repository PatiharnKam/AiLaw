package app

const (
	SUCCESS_CODE = "00000"
	SUCCESS_MSG  = "SUCCESS"
)

const (
	InvalidRequestErrorCode           = "10000"
	UserPromptLengthExceededErrorCode = "10001"
	QuotaExceededErrorCode            = "10002"
	UnauthorizedErrorCode             = "10003"
	InternalServerErrorCode           = "99999"

	UserPromptLengthExceededErrorMessage = "user prompt length exceeded"
	QuotaExceededErrorMessage            = "quota exceeded"
	UnauthorizedErrorMessage             = "unauthorized access"
	InvalidRequestErrorMessage           = "invalid request"
	InternalServerErrorMessage           = "internal server error"
	ActionLogout                         = "logout"
	ActionRefresh                        = "refresh"
)
