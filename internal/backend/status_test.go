package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBackendStatus_String_AllValues(t *testing.T) {
	assert.Equal(t, "Unknown", StatusUnknown.String())
	assert.Equal(t, "Available", StatusAvailable.String())
	assert.Equal(t, "Locked", StatusLocked.String())
	assert.Equal(t, "NotSignedIn", StatusNotSignedIn.String())
	assert.Equal(t, "Unavailable", StatusUnavailable.String())
}

func TestBackendStatus_String_UnknownValue(t *testing.T) {
	outOfRange := BackendStatus(99)
	assert.Equal(t, "Unknown", outOfRange.String())
}
