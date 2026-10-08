package tables

import (
	"errors"
	"strings"
	"testing"

	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
)

func message(t *testing.T, err error, language string) string {
	t.Helper()
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error %v is not an AppError", err)
	}
	return appErr.Message(language)
}

// The checks name the field by its caption in the user's language.
func TestChecksAreTranslated(t *testing.T) {
	err := CheckMaxLength("Customer", "post_code", strings.Repeat("1", 21), 20)
	if got := message(t, err, "en-US"); got != "Post Code cannot exceed 20 characters" {
		t.Errorf("en-US: %q", got)
	}
	if got := message(t, err, "nb-NO"); !strings.Contains(got, "20 tegn") || strings.Contains(got, "post_code") {
		t.Errorf("nb-NO: %q, want a Norwegian message with the field caption", got)
	}
	if got := message(t, CheckRequired("Payment Terms", "code", true), "en-US"); got != "Code is required" {
		t.Errorf("required: %q", got)
	}
	if got := message(t, CheckRange("Job_Queue", "minutes_between_run", 0, 1, 1440), "en-US"); !strings.Contains(got, "between 1 and 1440") {
		t.Errorf("range: %q", got)
	}
}

// Lengths count characters, as the database's VARCHAR does: 50 Norwegian letters fit in 50.
func TestCheckMaxLengthCountsCharacters(t *testing.T) {
	name := strings.Repeat("ø", 50)
	if err := CheckMaxLength("Customer", "name", name, 50); err != nil {
		t.Errorf("50 × ø (100 bytes) rejected for a 50-character field: %v", err)
	}
	if err := CheckMaxLength("Customer", "name", name+"x", 50); err == nil {
		t.Error("51 characters accepted for a 50-character field")
	}
	if err := CheckRequired("Customer", "no", false); err != nil {
		t.Errorf("a field with a value: %v", err)
	}
}
