package codeunits

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
