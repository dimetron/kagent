package errors

import "fmt"

// AppError represents an application-level error with a code and optional cause
type AppError struct {
	Code    string
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

// New creates a new AppError
func New(code, message string, cause error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Error codes
const (
	ErrCodeSessionCreate     = "SESSION_CREATE_FAILED"
	ErrCodeSessionGet        = "SESSION_GET_FAILED"
	ErrCodeSessionDelete     = "SESSION_DELETE_FAILED"
	ErrCodeAgentConfig       = "AGENT_CONFIG_INVALID"
	ErrCodeToolExecution     = "TOOL_EXECUTION_FAILED"
	ErrCodeConversion        = "CONVERSION_FAILED"
	ErrCodeExecutorFailed    = "EXECUTOR_FAILED"
	ErrCodePathManagement    = "PATH_MANAGEMENT_FAILED"
	ErrCodeFileOperation     = "FILE_OPERATION_FAILED"
	ErrCodeSkillNotFound     = "SKILL_NOT_FOUND"
	ErrCodeArtifactTooLarge  = "ARTIFACT_TOO_LARGE"
	ErrCodeInvalidInput      = "INVALID_INPUT"
	ErrCodeAuthFailed        = "AUTH_FAILED"
)
