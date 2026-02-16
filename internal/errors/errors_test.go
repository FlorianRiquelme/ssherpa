package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackendError_Error(t *testing.T) {
	err := &BackendError{
		Op:      "GetServer",
		Backend: "mock",
		Err:     ErrServerNotFound,
	}

	msg := err.Error()
	assert.Contains(t, msg, "GetServer")
	assert.Contains(t, msg, "mock")
	assert.Contains(t, msg, "server not found")
}

func TestBackendError_Unwrap(t *testing.T) {
	inner := ErrServerNotFound
	err := &BackendError{
		Op:      "GetServer",
		Backend: "mock",
		Err:     inner,
	}

	// Unwrap returns inner error
	assert.Equal(t, inner, err.Unwrap())

	// errors.Is works through the chain
	assert.True(t, Is(err, ErrServerNotFound))
	assert.False(t, Is(err, ErrProjectNotFound))
}

func TestBackendError_As(t *testing.T) {
	err := &BackendError{
		Op:      "CreateServer",
		Backend: "sshconfig",
		Err:     ErrDuplicateID,
	}

	// Wrap it further
	wrapped := fmt.Errorf("operation failed: %w", err)

	var be *BackendError
	require.True(t, As(wrapped, &be))
	assert.Equal(t, "CreateServer", be.Op)
	assert.Equal(t, "sshconfig", be.Backend)
}

func TestSentinelErrors_AreDistinct(t *testing.T) {
	sentinels := []error{
		ErrBackendUnavailable,
		ErrConfigNotFound,
		ErrServerNotFound,
		ErrProjectNotFound,
		ErrCredentialNotFound,
		ErrReadOnlyBackend,
		ErrDuplicateID,
		ErrValidation,
	}

	// Each sentinel must be identifiable via errors.Is
	for i, s := range sentinels {
		assert.True(t, Is(s, s), "sentinel %d should match itself", i)

		// Each must be distinct from all others
		for j, other := range sentinels {
			if i != j {
				assert.False(t, Is(s, other), "sentinel %d should not match sentinel %d", i, j)
			}
		}
	}
}

func TestReExportedFunctions(t *testing.T) {
	// New creates a new error
	err := New("test error")
	assert.EqualError(t, err, "test error")

	// Is matches correctly
	assert.True(t, Is(err, err))

	// Unwrap on a non-wrapped error returns nil
	assert.Nil(t, Unwrap(err))

	// Unwrap on a wrapped error returns inner
	wrapped := fmt.Errorf("outer: %w", err)
	assert.Equal(t, err, Unwrap(wrapped))
}
