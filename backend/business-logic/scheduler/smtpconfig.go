package scheduler

import (
	"strconv"

	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/mail"
)

// currentMailer returns the mail sender to use for notifications. A mailer set
// on the struct (test injection) takes precedence; otherwise configuration is
// loaded from the global SMTP_Setup table. When no enabled setup exists, a
// disabled mailer is returned so Send is a safe no-op.
func (s *Scheduler) currentMailer() mail.Sender {
	if s.mailer != nil {
		return s.mailer
	}
	cfg := s.loadSMTPConfig()
	return mail.NewSMTPMailer(cfg)
}

// loadSMTPConfig reads the single SMTP_Setup record (a BC-style setup table with
// a blank primary key) and builds a mail.Config. If the record is missing or not
// enabled, a disabled config is returned.
func (s *Scheduler) loadSMTPConfig() mail.Config {
	var setup tables.SMTPSetup
	// SMTP_Setup is a global setup table; the company argument is ignored and the
	// single record is keyed by a blank primary key.
	setup.InitWithDBType(s.db, "", s.dbType)

	if setup.Get("") && setup.Enabled {
		port := setup.Smtp_server_port
		if port <= 0 {
			port = 587
		}
		return mail.Config{
			Enabled:  true,
			Host:     setup.Smtp_server.String(),
			Port:     strconv.Itoa(port),
			Username: setup.User_id.String(),
			Password: setup.Password.String(),
			From:     setup.From_address.String(),
		}
	}
	return mail.Config{Enabled: false}
}
