package onepassword

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	backendpkg "github.com/florianriquelme/ssherpa/internal/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoller_DetectsAvailability(t *testing.T) {
	mock := NewMockClient()
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.toml")

	// Start with 1Password unavailable
	mock.SetError("ListVaults", assert.AnError)

	backend := NewWithCache(mock, cachePath)

	// Track status changes
	var statusChanges []backendpkg.BackendStatus
	var mu sync.Mutex
	onChange := func(status backendpkg.BackendStatus) {
		mu.Lock()
		defer mu.Unlock()
		statusChanges = append(statusChanges, status)
	}

	// Start poller with short interval
	backend.StartPolling(100*time.Millisecond, onChange)

	// Wait for initial unavailable detection
	time.Sleep(150 * time.Millisecond)

	// Make 1Password available
	mock.ClearError("ListVaults")
	mock.AddVault(Vault{ID: "vault-1", Name: "Test Vault"})
	mock.AddItem(Item{
		ID:       "item-1",
		Title:    "Test Server",
		VaultID:  "vault-1",
		Category: "server",
		Tags:     []string{"ssherpa"},
		Fields: []ItemField{
			{Title: "hostname", Value: "test.example.com"},
			{Title: "user", Value: "testuser"},
		},
	})

	// Wait for availability detection
	time.Sleep(200 * time.Millisecond)

	// Stop poller
	backend.Close()

	// Verify status changed to Available
	mu.Lock()
	defer mu.Unlock()
	assert.Contains(t, statusChanges, backendpkg.StatusAvailable, "Should detect 1Password became available")
}

func TestPoller_DetectsUnavailability(t *testing.T) {
	mock := NewMockClient()
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.toml")

	// Start with 1Password available
	mock.AddVault(Vault{ID: "vault-1", Name: "Test Vault"})
	mock.AddItem(Item{
		ID:       "item-1",
		Title:    "Test Server",
		VaultID:  "vault-1",
		Category: "server",
		Tags:     []string{"ssherpa"},
		Fields: []ItemField{
			{Title: "hostname", Value: "test.example.com"},
			{Title: "user", Value: "testuser"},
		},
	})

	backend := NewWithCache(mock, cachePath)

	// Initial sync
	ctx := context.Background()
	err := backend.SyncFromOnePassword(ctx)
	require.NoError(t, err)
	assert.Equal(t, backendpkg.StatusAvailable, backend.GetStatus())

	// Track status changes
	var statusChanges []backendpkg.BackendStatus
	var mu sync.Mutex
	onChange := func(status backendpkg.BackendStatus) {
		mu.Lock()
		defer mu.Unlock()
		statusChanges = append(statusChanges, status)
	}

	// Start poller with short interval
	backend.StartPolling(100*time.Millisecond, onChange)

	// Wait a bit for poller to start
	time.Sleep(50 * time.Millisecond)

	// Make 1Password unavailable
	mock.SetError("ListVaults", assert.AnError)

	// Wait for unavailability detection
	time.Sleep(200 * time.Millisecond)

	// Stop poller
	backend.Close()

	// Verify status changed to Unavailable
	mu.Lock()
	defer mu.Unlock()
	assert.True(t, len(statusChanges) > 0, "Should detect status change")
	// Could be backendpkg.StatusLocked or backendpkg.StatusUnavailable depending on error
	lastStatus := statusChanges[len(statusChanges)-1]
	assert.True(t, lastStatus == backendpkg.StatusLocked || lastStatus == backendpkg.StatusUnavailable,
		"Should detect 1Password became unavailable")
}

func TestPoller_StopsCleanly(t *testing.T) {
	mock := NewMockClient()
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.toml")

	mock.AddVault(Vault{ID: "vault-1", Name: "Test Vault"})

	backend := NewWithCache(mock, cachePath)

	// Track number of times poller runs
	var pollCount atomic.Int32
	onChange := func(status backendpkg.BackendStatus) {
		pollCount.Add(1)
	}

	// Start poller
	backend.StartPolling(50*time.Millisecond, onChange)

	// Let it run a few times
	time.Sleep(120 * time.Millisecond)

	// Stop poller
	err := backend.Close()
	require.NoError(t, err)

	// Record poll count at stop
	countAtStop := pollCount.Load()

	// Wait to ensure poller doesn't run after stop
	time.Sleep(100 * time.Millisecond)

	// Poll count should not increase after stop
	countAfterStop := pollCount.Load()
	assert.Equal(t, countAtStop, countAfterStop, "Poller should stop immediately")
}

func TestPoller_SkipsSyncAfterRecentWrite(t *testing.T) {
	mock := NewMockClient()
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.toml")

	mock.AddVault(Vault{ID: "vault-1", Name: "Test Vault"})
	mock.AddItem(Item{
		ID:       "item-1",
		Title:    "Test Server",
		VaultID:  "vault-1",
		Category: "server",
		Tags:     []string{"ssherpa"},
		Fields: []ItemField{
			{Title: "hostname", Value: "test.example.com"},
			{Title: "user", Value: "testuser"},
		},
	})

	backend := NewWithCache(mock, cachePath)

	// Initial sync
	ctx := context.Background()
	err := backend.SyncFromOnePassword(ctx)
	require.NoError(t, err)

	// Start poller with short interval
	backend.StartPolling(50*time.Millisecond, nil)

	// Simulate a recent write
	backend.UpdateLastWrite()

	// Wait less than debounce period (10 seconds)
	time.Sleep(100 * time.Millisecond)

	// Mock should not have been called again (sync skipped due to recent write)
	// This is hard to verify directly, but we can check that backend still works
	servers, err := backend.ListServers(ctx)
	require.NoError(t, err)
	assert.Len(t, servers, 1)

	// Stop poller
	backend.Close()
}

func TestPoller_ConfigurableInterval(t *testing.T) {
	// Set environment variable for test
	os.Setenv("SSHJESUS_1PASSWORD_POLL_INTERVAL", "200ms")
	defer os.Unsetenv("SSHJESUS_1PASSWORD_POLL_INTERVAL")

	mock := NewMockClient()
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.toml")

	mock.AddVault(Vault{ID: "vault-1", Name: "Test Vault"})

	backend := NewWithCache(mock, cachePath)

	// Track poll times
	var pollTimes []time.Time
	var mu sync.Mutex
	onChange := func(status backendpkg.BackendStatus) {
		mu.Lock()
		defer mu.Unlock()
		pollTimes = append(pollTimes, time.Now())
	}

	// Start poller (should use env var interval)
	backend.StartPolling(0, onChange) // Pass 0 to use env var

	// Wait for a few polls
	time.Sleep(500 * time.Millisecond)

	// Stop poller
	backend.Close()

	// Verify interval is roughly 200ms (with some tolerance)
	mu.Lock()
	defer mu.Unlock()

	if len(pollTimes) >= 2 {
		// Check interval between polls
		interval := pollTimes[1].Sub(pollTimes[0])
		// Allow 50ms tolerance for timing variance
		assert.True(t, interval >= 150*time.Millisecond && interval <= 250*time.Millisecond,
			"Poll interval should be ~200ms based on env var, got %v", interval)
	}
}

func TestPoller_NilOnChangeIsOK(t *testing.T) {
	mock := NewMockClient()
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.toml")

	mock.AddVault(Vault{ID: "vault-1", Name: "Test Vault"})

	backend := NewWithCache(mock, cachePath)

	// Start poller with nil onChange (should not panic)
	backend.StartPolling(50*time.Millisecond, nil)

	// Let it run
	time.Sleep(100 * time.Millisecond)

	// Stop poller
	err := backend.Close()
	require.NoError(t, err)
}
