package codeunits

import (
	"time"

	"github.com/hansjlachmann/openerp/backend/foundation/session"
)

// CurrentUserID returns the current session user ID
func CurrentUserID() string {
	sess := session.GetCurrent()
	if sess != nil {
		return sess.GetUserID()
	}
	return ""
}

// CurrentCompany returns the current session company name
func CurrentCompany() string {
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
