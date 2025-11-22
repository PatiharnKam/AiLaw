package app

const (
	SUCCESS_CODE = "00000"
	SUCCESS_MSG  = "SUCCESS"
)

const (
	InvalidRequestErrorCode = "10000"
	UnauthorizedErrorCode   = "10003"
	InternalServerErrorCode = "99999"

	UnauthorizedErrorMessage   = "unauthorizrd access"
	InvalidRequestErrorMessage = "invalid request"
	InternalServerErrorMessage = "internal server error"
	ActionLogout               = "logout"
	ActionRefresh              = "refresh"
)
