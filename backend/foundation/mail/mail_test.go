package mail

import (
	"strings"
	"testing"
)

func TestDisabledMailerIsNoop(t *testing.T) {
	m := NewSMTPMailer(Config{Enabled: false, Host: "smtp.example.com", From: "a@b.com"})
	if m.Enabled() {
		t.Fatal("mailer should be disabled")
	}
	if err := m.Send("x@y.com", "subj", "body"); err != nil {
		t.Fatalf("disabled Send should be a no-op, got %v", err)
	}
}

func TestEnabledRequiresHostAndFrom(t *testing.T) {
	if NewSMTPMailer(Config{Enabled: true, From: "a@b.com"}).Enabled() {
		t.Error("missing host should not be enabled")
	}
	if NewSMTPMailer(Config{Enabled: true, Host: "h"}).Enabled() {
		t.Error("missing from should not be enabled")
	}
	if !NewSMTPMailer(Config{Enabled: true, Host: "h", From: "a@b.com"}).Enabled() {
		t.Error("host+from+enabled should be enabled")
	}
}

func TestBuildMessageSanitizesHeaders(t *testing.T) {
	// A subject with CRLF must not be able to inject an extra header line.
	msg := string(buildMessage("from@x.com", "to@x.com", "hi\r\nBcc: evil@x.com", "body"))
	if strings.Contains(msg, "\r\nBcc:") {
		t.Errorf("header injection not prevented (Bcc appears as a header line):\n%s", msg)
	}
	if !strings.Contains(msg, "Subject: hiBcc: evil@x.com") {
		t.Errorf("subject not sanitized as expected:\n%s", msg)
	}
	if !strings.HasSuffix(msg, "\r\nbody") {
		t.Errorf("body not appended correctly:\n%s", msg)
	}
}
