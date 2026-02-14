package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors for common failure modes.
var (
	ErrBackendUnavailable = errors.New("backend unavailable")
	ErrConfigNotFound     = errors.New("config file not found")
	ErrServerNotFound     = errors.New("server not found")
	ErrProjectNotFound    = errors.New("project not found")
	ErrCredentialNotFound = errors.New("credential not found")
	ErrReadOnlyBackend    = errors.New("backend does not support write operations")
	ErrDuplicateID        = errors.New("duplicate ID")
	ErrValidation         = errors.New("validation error")
)

// BackendError wraps errors with operation and backend context.
// Implements Unwrap() for error chain inspection via errors.Is/As.
type BackendError struct {
	Op      string // operation that failed (e.g., "GetServer", "CreateServer")
	Backend string // backend name (e.g., "mock", "sshconfig", "onepassword")
	Err     error  // underlying cause
}

// Error returns a formatted error message with operation and backend context.
func (e *BackendError) Error() string {
	return fmt.Sprintf("%s: %s backend: %v", e.Op, e.Backend, e.Err)
}

// Unwrap returns the underlying error for error chain inspection.
func (e *BackendError) Unwrap() error {
	return e.Err
}

// Re-export standard library error functions for convenience.
// Allows callers to import only sshjesus/internal/errors instead of both packages.
var (
	Is     = errors.Is
	As     = errors.As
	New    = errors.New
	Unwrap = errors.Unwrap
)
