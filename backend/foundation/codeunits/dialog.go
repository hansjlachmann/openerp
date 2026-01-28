package codeunits

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ProgressEvent represents a progress update
type ProgressEvent struct {
	JobID      string `json:"job_id"`
	Field      int    `json:"field"`      // Field number (1-based, like NAV)
	Value      int    `json:"value"`      // Current value (0-100 for progress)
	Message    string `json:"message"`    // Optional message
	Completed  bool   `json:"completed"`  // Job completed
	Error      string `json:"error"`      // Error message if failed
	Timestamp  int64  `json:"timestamp"`
}

// Dialog represents a NAV-style dialog with progress bar support
// Usage:
//   dialog := OpenDialog("Processing @1@@@@@@@@@@@")
//   defer dialog.Close()
//   for i := 1; i <= 100; i++ {
//       dialog.Update(1, i)
//   }
type Dialog struct {
	jobID     string
	template  string
	events    chan ProgressEvent
	ctx       context.Context
	cancel    context.CancelFunc
	closed    bool
	mu        sync.Mutex
}

// Update sends a progress update for a field
// field is 1-based (like NAV @1@, @2@, etc.)
// value is typically 0-100 for percentage
func (d *Dialog) Update(field int, value int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return
	}

	select {
	case d.events <- ProgressEvent{
		JobID:     d.jobID,
		Field:     field,
		Value:     value,
		Timestamp: time.Now().UnixMilli(),
	}:
	case <-d.ctx.Done():
	default:
		// Non-blocking, drop if buffer full
	}
}

// UpdateWithMessage sends a progress update with a message
func (d *Dialog) UpdateWithMessage(field int, value int, message string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return
	}

	select {
	case d.events <- ProgressEvent{
		JobID:     d.jobID,
		Field:     field,
		Value:     value,
		Message:   message,
		Timestamp: time.Now().UnixMilli(),
	}:
	case <-d.ctx.Done():
	default:
	}
}

// Close closes the dialog and signals completion
func (d *Dialog) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return
	}
	d.closed = true

	select {
	case d.events <- ProgressEvent{
		JobID:     d.jobID,
		Completed: true,
		Timestamp: time.Now().UnixMilli(),
	}:
	case <-d.ctx.Done():
	default:
	}

	// Give SSE clients time to receive the completion event
	time.AfterFunc(500*time.Millisecond, func() {
		d.cancel()
		jobRegistry.Remove(d.jobID)
	})
}

// CloseWithError closes the dialog with an error
func (d *Dialog) CloseWithError(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return
	}
	d.closed = true

	select {
	case d.events <- ProgressEvent{
		JobID:     d.jobID,
		Completed: true,
		Error:     err.Error(),
		Timestamp: time.Now().UnixMilli(),
	}:
	case <-d.ctx.Done():
	default:
	}

	time.AfterFunc(500*time.Millisecond, func() {
		d.cancel()
		jobRegistry.Remove(d.jobID)
	})
}

// JobID returns the job ID for this dialog
func (d *Dialog) JobID() string {
	return d.jobID
}

// Events returns the channel for progress events (used by SSE handler)
func (d *Dialog) Events() <-chan ProgressEvent {
	return d.events
}

// Context returns the context for this dialog
func (d *Dialog) Context() context.Context {
	return d.ctx
}

// JobRegistry tracks running jobs with dialogs
type JobRegistry struct {
	jobs map[string]*Dialog
	mu   sync.RWMutex
}

var jobRegistry = &JobRegistry{
	jobs: make(map[string]*Dialog),
}

// GetJobRegistry returns the global job registry
func GetJobRegistry() *JobRegistry {
	return jobRegistry
}

// Register adds a job to the registry
func (r *JobRegistry) Register(d *Dialog) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs[d.jobID] = d
}

// Get retrieves a job by ID
func (r *JobRegistry) Get(jobID string) (*Dialog, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.jobs[jobID]
	return d, ok
}

// Remove removes a job from the registry
func (r *JobRegistry) Remove(jobID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.jobs, jobID)
}

// OpenDialog creates a new dialog with progress bar support
// template follows NAV convention: @1@@@@@@@@@@@ for field 1 progress bar
// Returns the dialog and its job ID
func OpenDialog(template string) *Dialog {
	ctx, cancel := context.WithCancel(context.Background())
	d := &Dialog{
		jobID:    uuid.New().String(),
		template: template,
		events:   make(chan ProgressEvent, 100), // Buffered channel
		ctx:      ctx,
		cancel:   cancel,
	}
	jobRegistry.Register(d)
	return d
}

// OpenDialogWithID creates a dialog with a specific job ID (for testing)
func OpenDialogWithID(template string, jobID string) *Dialog {
	ctx, cancel := context.WithCancel(context.Background())
	d := &Dialog{
		jobID:    jobID,
		template: template,
		events:   make(chan ProgressEvent, 100),
		ctx:      ctx,
		cancel:   cancel,
	}
	jobRegistry.Register(d)
	return d
}
