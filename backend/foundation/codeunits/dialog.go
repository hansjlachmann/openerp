package codeunits

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ProgressEvent represents a progress update or dialog event
type ProgressEvent struct {
	JobID      string                 `json:"job_id"`
	Field      int                    `json:"field"`                  // Field number (1-based, like NAV)
	Value      int                    `json:"value"`                  // Current value (0-100 for progress)
	Message    string                 `json:"message"`                // Optional message
	Completed  bool                   `json:"completed"`              // Job completed
	Error      string                 `json:"error"`                  // Error message if failed
	Timestamp  int64                  `json:"timestamp"`
	EventType  string                 `json:"event_type,omitempty"`   // "progress", "confirm", etc.
	ConfirmID  string                 `json:"confirm_id,omitempty"`   // Unique ID for confirm dialogs
	Data       map[string]interface{} `json:"data,omitempty"`         // Result data (e.g., PDF base64)
}

// InputField describes a single input field for RequestInput dialogs
type InputField struct {
	Name     string `json:"name"`               // Field key (e.g. "date")
	Label    string `json:"label"`              // Display label
	Type     string `json:"type"`               // "date", "text", "number"
	Required bool   `json:"required,omitempty"` // Whether the field is required
	Default  string `json:"default,omitempty"`  // Default value
}

// Dialog represents a NAV-style dialog with progress bar support
// Usage:
//   dialog := OpenDialog("Processing @1@@@@@@@@@@@")
//   defer dialog.Close()
//   for i := 1; i <= 100; i++ {
//       dialog.Update(1, i)
//   }
type Dialog struct {
	jobID          string
	template       string
	events         chan ProgressEvent
	ctx            context.Context
	cancel         context.CancelFunc
	closed         bool
	mu             sync.Mutex
	pendingConfirm map[string]chan bool              // Map of confirm ID to response channel
	confirmMu      sync.Mutex
	pendingInput   map[string]chan map[string]string  // Map of confirm ID to input response channel
	inputMu        sync.Mutex
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

// CloseWithData closes the dialog with result data (e.g., PDF base64)
func (d *Dialog) CloseWithData(data map[string]interface{}, message string) {
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
		Message:   message,
		Data:      data,
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

// Cancel cancels the running job
func (d *Dialog) Cancel() {
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
		Error:     "Cancelled by user",
		EventType: "cancelled",
		Timestamp: time.Now().UnixMilli(),
	}:
	case <-d.ctx.Done():
	default:
	}

	// Cancel the context immediately to stop the codeunit
	d.cancel()
	jobRegistry.Remove(d.jobID)
}

// Events returns the channel for progress events (used by SSE handler)
func (d *Dialog) Events() <-chan ProgressEvent {
	return d.events
}

// Context returns the context for this dialog
func (d *Dialog) Context() context.Context {
	return d.ctx
}

// Confirm shows a confirmation dialog and waits for user response
// Returns true if user clicks Yes, false if No
// Similar to NAV/BC: Confirm('Are you sure?')
func (d *Dialog) Confirm(message string) bool {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return false
	}
	d.mu.Unlock()

	// Create unique confirm ID and response channel
	confirmID := uuid.New().String()
	responseChan := make(chan bool, 1)

	// Register the pending confirm
	d.confirmMu.Lock()
	if d.pendingConfirm == nil {
		d.pendingConfirm = make(map[string]chan bool)
	}
	d.pendingConfirm[confirmID] = responseChan
	d.confirmMu.Unlock()

	// Clean up on completion
	defer func() {
		d.confirmMu.Lock()
		delete(d.pendingConfirm, confirmID)
		d.confirmMu.Unlock()
	}()

	// Send confirm event to frontend
	select {
	case d.events <- ProgressEvent{
		JobID:     d.jobID,
		EventType: "confirm",
		ConfirmID: confirmID,
		Message:   message,
		Timestamp: time.Now().UnixMilli(),
	}:
	case <-d.ctx.Done():
		return false
	}

	// Wait for response
	select {
	case response := <-responseChan:
		return response
	case <-d.ctx.Done():
		return false
	case <-time.After(5 * time.Minute): // Timeout after 5 minutes
		return false
	}
}

// RespondToConfirm handles a confirm response from the frontend
func (d *Dialog) RespondToConfirm(confirmID string, response bool) bool {
	d.confirmMu.Lock()
	defer d.confirmMu.Unlock()

	if ch, ok := d.pendingConfirm[confirmID]; ok {
		select {
		case ch <- response:
			return true
		default:
			return false
		}
	}
	return false
}

// RequestInput shows an input dialog with arbitrary fields and waits for user response
// Returns a map of field name → value, or nil on cancel/timeout
func (d *Dialog) RequestInput(title string, fields []InputField) map[string]string {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.mu.Unlock()

	// Create unique confirm ID and response channel
	confirmID := uuid.New().String()
	responseChan := make(chan map[string]string, 1)

	// Register the pending input
	d.inputMu.Lock()
	if d.pendingInput == nil {
		d.pendingInput = make(map[string]chan map[string]string)
	}
	d.pendingInput[confirmID] = responseChan
	d.inputMu.Unlock()

	// Clean up on completion
	defer func() {
		d.inputMu.Lock()
		delete(d.pendingInput, confirmID)
		d.inputMu.Unlock()
	}()

	// Send request_input event to frontend
	select {
	case d.events <- ProgressEvent{
		JobID:     d.jobID,
		EventType: "request_input",
		ConfirmID: confirmID,
		Message:   title,
		Timestamp: time.Now().UnixMilli(),
		Data: map[string]interface{}{
			"fields": fields,
		},
	}:
	case <-d.ctx.Done():
		return nil
	}

	// Wait for response
	select {
	case values := <-responseChan:
		return values
	case <-d.ctx.Done():
		return nil
	case <-time.After(5 * time.Minute):
		return nil
	}
}

// RespondToInput handles an input dialog response from the frontend
func (d *Dialog) RespondToInput(confirmID string, values map[string]string) bool {
	d.inputMu.Lock()
	defer d.inputMu.Unlock()

	if ch, ok := d.pendingInput[confirmID]; ok {
		select {
		case ch <- values:
			return true
		default:
			return false
		}
	}
	return false
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
