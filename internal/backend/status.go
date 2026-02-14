package backend

// BackendStatus represents the availability status of a backend.
// This is defined at the backend level (not in onepassword package)
// to avoid import cycles when TUI needs to reference status.
type BackendStatus int

const (
	StatusUnknown     BackendStatus = iota // Initial state before first check
	StatusAvailable                        // Backend is unlocked and responsive
	StatusLocked                           // Backend is running but locked
	StatusUnavailable                      // Backend not running or SDK error
)

// String returns the string representation of the status.
func (s BackendStatus) String() string {
	switch s {
	case StatusUnknown:
		return "Unknown"
	case StatusAvailable:
		return "Available"
	case StatusLocked:
		return "Locked"
	case StatusUnavailable:
		return "Unavailable"
	default:
		return "Unknown"
	}
}
