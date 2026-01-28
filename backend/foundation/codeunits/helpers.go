package codeunits

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/hansjlachmann/openerp/backend/foundation/session"
)

// currentDialogs holds the dialog for each goroutine
var currentDialogs sync.Map

// goroutineContext holds per-goroutine session context
type goroutineContext struct {
	UserID   string
	UserName string
	Company  string
	Language string
}

// currentContexts holds the context for each goroutine
var currentContexts sync.Map

// SetCurrentContext sets the session context for the current goroutine
// This should be called at the start of a goroutine that runs codeunits
func SetCurrentContext(userID, userName, company, language string) {
	ctx := &goroutineContext{
		UserID:   userID,
		UserName: userName,
		Company:  company,
		Language: language,
	}
	currentContexts.Store(getGoroutineID(), ctx)
}

// GetCurrentContext returns the context for the current goroutine
func GetCurrentContext() *goroutineContext {
	if ctx, ok := currentContexts.Load(getGoroutineID()); ok {
		return ctx.(*goroutineContext)
	}
	return nil
}

// ClearCurrentContext clears the context for the current goroutine
func ClearCurrentContext() {
	currentContexts.Delete(getGoroutineID())
}

// SetCurrentDialog sets the dialog for the current goroutine
func SetCurrentDialog(d *Dialog) {
	currentDialogs.Store(getGoroutineID(), d)
}

// GetCurrentDialog returns the dialog for the current goroutine
func GetCurrentDialog() *Dialog {
	if d, ok := currentDialogs.Load(getGoroutineID()); ok {
		return d.(*Dialog)
	}
	return nil
}

// ClearCurrentDialog clears the dialog for the current goroutine
func ClearCurrentDialog() {
	currentDialogs.Delete(getGoroutineID())
}

// getGoroutineID returns the ID of the current goroutine
func getGoroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	// Stack trace format: "goroutine 123 [running]:"
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

// CurrentUserID returns the current user ID
// First checks goroutine-specific context, then falls back to global session
func CurrentUserID() string {
	// Check goroutine-specific context first
	if ctx := GetCurrentContext(); ctx != nil {
		return ctx.UserID
	}
	// Fall back to global session
	sess := session.GetCurrent()
	if sess != nil {
		return sess.GetUserID()
	}
	return ""
}

// CurrentCompany returns the current company name
// First checks goroutine-specific context, then falls back to global session
func CurrentCompany() string {
	// Check goroutine-specific context first
	if ctx := GetCurrentContext(); ctx != nil {
		return ctx.Company
	}
	// Fall back to global session
	sess := session.GetCurrent()
	if sess != nil {
		return sess.GetCompany()
	}
	return ""
}

// CurrentDateTime returns the current date and time
func CurrentDateTime() time.Time {
	return time.Now()
}

// CurrentDate returns the current date (time set to midnight)
func CurrentDate() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// CurrentTime returns the current time (date set to zero)
func CurrentTime() time.Time {
	now := time.Now()
	return time.Date(0, 1, 1, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location())
}

// Message returns a result that displays an info dialog
func Message(text string) Result {
	return Result{
		Success: true,
		Dialog: &DialogResult{
			Title:   "Message",
			Message: text,
			Type:    "info",
		},
	}
}

// Error returns a result that displays an error dialog
func Error(text string) Result {
	return Result{
		Success: false,
		Dialog: &DialogResult{
			Title:   "Error",
			Message: text,
			Type:    "error",
		},
	}
}

// Confirm shows a confirmation dialog and waits for user response
// Returns true if user clicks Yes, false if No
// Similar to NAV/BC: Confirm('Are you sure?')
// Note: This only works in async codeunits (UsesProgress() = true)
func Confirm(message string) bool {
	dialog := GetCurrentDialog()
	if dialog == nil {
		// No dialog context - can't show confirm
		// Return false as safe default
		return false
	}
	return dialog.Confirm(message)
}
