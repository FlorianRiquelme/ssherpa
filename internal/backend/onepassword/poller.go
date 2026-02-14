package onepassword

import (
	"context"
	"os"
	"sync"
	"time"
)

// Poller periodically checks 1Password availability and auto-recovers when it becomes available.
type Poller struct {
	backend  *Backend
	interval time.Duration
	ticker   *time.Ticker
	stopCh   chan struct{}
	wg       sync.WaitGroup
	onChange func(BackendStatus) // Callback when status changes (for TUI notification)
}

// NewPoller creates a new poller for the backend.
// interval: how often to poll (use 0 to read from SSHJESUS_1PASSWORD_POLL_INTERVAL env var, defaults to 5s)
// onChange: optional callback invoked when status changes (nil = no callback)
func NewPoller(backend *Backend, interval time.Duration, onChange func(BackendStatus)) *Poller {
	// Check environment variable for interval override
	if interval == 0 {
		interval = 5 * time.Second // default
		if envInterval := os.Getenv("SSHJESUS_1PASSWORD_POLL_INTERVAL"); envInterval != "" {
			if parsed, err := time.ParseDuration(envInterval); err == nil {
				interval = parsed
			}
		}
	}

	return &Poller{
		backend:  backend,
		interval: interval,
		stopCh:   make(chan struct{}),
		onChange: onChange,
	}
}

// Start begins polling in a background goroutine.
// Returns immediately - use Stop() to halt polling.
func (p *Poller) Start() {
	p.ticker = time.NewTicker(p.interval)
	p.wg.Add(1)

	go func() {
		defer p.wg.Done()
		defer p.ticker.Stop()

		for {
			select {
			case <-p.stopCh:
				return
			case <-p.ticker.C:
				p.poll()
			}
		}
	}()
}

// poll performs a single poll operation.
func (p *Poller) poll() {
	// Check if there was a recent write (within last 10 seconds)
	p.backend.mu.RLock()
	lastWrite := p.backend.lastWrite
	p.backend.mu.RUnlock()

	if !lastWrite.IsZero() && time.Since(lastWrite) < 10*time.Second {
		// Skip sync - too soon after write
		return
	}

	// Get current status before sync
	oldStatus := p.backend.GetStatus()

	// Attempt sync with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = p.backend.SyncFromOnePassword(ctx)
	// Error is OK - status will be set appropriately by SyncFromOnePassword

	// Check if status changed
	newStatus := p.backend.GetStatus()
	if newStatus != oldStatus {
		// Status changed - notify if callback provided
		if p.onChange != nil {
			p.onChange(newStatus)
		}
	}
}

// Stop halts the poller and waits for the goroutine to exit.
func (p *Poller) Stop() {
	close(p.stopCh)
	p.wg.Wait()
}

// StartPolling starts a background poller for this backend.
// This is a convenience method that creates and starts a poller.
func (b *Backend) StartPolling(interval time.Duration, onChange func(BackendStatus)) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Stop existing poller if any
	if b.poller != nil {
		b.poller.Stop()
	}

	// Create and start new poller
	b.poller = NewPoller(b, interval, onChange)
	b.poller.Start()
}

// UpdateLastWrite updates the last write timestamp to prevent sync loops.
// This should be called after CreateServer, UpdateServer, or DeleteServer.
func (b *Backend) UpdateLastWrite() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lastWrite = time.Now()
}
