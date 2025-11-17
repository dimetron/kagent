package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	err := New(ErrCodeToolExecution, "tool failed", nil)

	assert.NotNil(t, err)
	assert.Equal(t, ErrCodeToolExecution, err.Code)
	assert.Equal(t, "tool failed", err.Message)
	assert.Nil(t, err.Cause)
}

func TestNew_WithCause(t *testing.T) {
	cause := errors.New("underlying error")
	err := New(ErrCodeToolExecution, "tool failed", cause)

	assert.NotNil(t, err)
	assert.Equal(t, ErrCodeToolExecution, err.Code)
	assert.Equal(t, "tool failed", err.Message)
	assert.Equal(t, cause, err.Cause)
}

func TestAppError_Error(t *testing.T) {
	err := New(ErrCodeToolExecution, "tool failed", nil)
	errorString := err.Error()

	assert.Contains(t, errorString, ErrCodeToolExecution)
	assert.Contains(t, errorString, "tool failed")
}

func TestAppError_Error_WithCause(t *testing.T) {
	cause := errors.New("underlying error")
	err := New(ErrCodeToolExecution, "tool failed", cause)
	errorString := err.Error()

	assert.Contains(t, errorString, ErrCodeToolExecution)
	assert.Contains(t, errorString, "tool failed")
	assert.Contains(t, errorString, "underlying error")
}

func TestErrorCodes(t *testing.T) {
	// Verify all error codes are unique and non-empty
	codes := []string{
		ErrCodeToolExecution,
		ErrCodeSessionCreate,
		ErrCodeSessionGet,
		ErrCodeInvalidInput,
		ErrCodeConversion,
		ErrCodeExecutorFailed,
		ErrCodeAgentConfig,
		ErrCodeAuthFailed,
	}

	seen := make(map[string]bool)
	for _, code := range codes {
		assert.NotEmpty(t, code)
		assert.False(t, seen[code], "duplicate error code: %s", code)
		seen[code] = true
	}
}

func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := New(ErrCodeToolExecution, "tool failed", cause)

	unwrapped := errors.Unwrap(err)
	assert.Equal(t, cause, unwrapped)
}

func TestAppError_Is(t *testing.T) {
	cause := errors.New("specific error")
	err := New(ErrCodeToolExecution, "tool failed", cause)

	// Should be able to check with errors.Is
	assert.True(t, errors.Is(err, cause))
}

func TestNew_DifferentCodes(t *testing.T) {
	tests := []struct {
		code    string
		message string
	}{
		{ErrCodeToolExecution, "tool execution failed"},
		{ErrCodeSessionCreate, "session creation failed"},
		{ErrCodeSessionGet, "session retrieval failed"},
		{ErrCodeInvalidInput, "invalid input provided"},
		{ErrCodeConversion, "conversion failed"},
		{ErrCodeExecutorFailed, "executor failed"},
		{ErrCodeAgentConfig, "agent configuration error"},
		{ErrCodeAuthFailed, "authentication failed"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			err := New(tt.code, tt.message, nil)
			assert.Equal(t, tt.code, err.Code)
			assert.Equal(t, tt.message, err.Message)
		})
	}
}

func TestAppError_NilCause(t *testing.T) {
	err := New(ErrCodeToolExecution, "tool failed", nil)
	errorString := err.Error()

	// Should not panic or include nil reference
	assert.NotEmpty(t, errorString)
	assert.NotContains(t, errorString, "<nil>")
	assert.NotContains(t, errorString, "nil")
}

func TestAppError_EmptyMessage(t *testing.T) {
	err := New(ErrCodeToolExecution, "", nil)

	// Should still create error with code
	assert.Equal(t, ErrCodeToolExecution, err.Code)
	assert.Equal(t, "", err.Message)

	errorString := err.Error()
	assert.Contains(t, errorString, ErrCodeToolExecution)
}
