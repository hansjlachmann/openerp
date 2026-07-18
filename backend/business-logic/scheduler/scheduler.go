// Package scheduler runs Job Queue jobs automatically on a recurring schedule.
//
// It is an in-process poller (no cron dependency): a ticker wakes periodically,
// acquires a DB-backed distributed lock (so only one instance runs across a
// multi-pod deployment), scans each company's Job_Queue for records that are
// Ready and due (next_start <= now), runs the referenced codeunit, reschedules
// the job per its recurrence, and optionally emails a notification.
package scheduler

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hansjlachmann/openerp/backend/business-logic/codeunits"
	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	fcodeunits "github.com/hansjlachmann/openerp/backend/foundation/codeunits"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	"github.com/hansjlachmann/openerp/backend/foundation/mail"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

// Scheduler polls Job_Queue tables and runs due jobs.
type Scheduler struct {
	db        database.Executor
	dbType    database.DBType
	companies func() ([]string, error)
	mailer    mail.Sender

	enabled  bool
	interval time.Duration
	lockTTL  time.Duration
	podID    string

	stop chan struct{}
	done chan struct{}
}

// New builds a Scheduler. Behavior is configured from the environment:
//
//	JOB_QUEUE_ENABLED        "false" to disable the scheduler (default enabled)
//	JOB_QUEUE_POLL_INTERVAL  poll interval in seconds (default 60, min 5)
//
// companies enumerates company names (e.g. companyMgr.ListCompanies). SMTP
// configuration for notifications is read from the global SMTP_Setup table.
func New(db database.Executor, dbType database.DBType, companies func() ([]string, error)) *Scheduler {
	enabled := !strings.EqualFold(os.Getenv("JOB_QUEUE_ENABLED"), "false")

	interval := 60 * time.Second
	if v := os.Getenv("JOB_QUEUE_POLL_INTERVAL"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs >= 5 {
			interval = time.Duration(secs) * time.Second
		}
	}

	podID := os.Getenv("HOSTNAME")
	if podID == "" {
		if h, err := os.Hostname(); err == nil {
			podID = h
		} else {
			podID = "scheduler"
		}
	}

	return &Scheduler{
		db:        db,
		dbType:    dbType,
		companies: companies,
		enabled:   enabled,
		interval:  interval,
		lockTTL:   5 * time.Minute,
		podID:     podID,
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
	}
}

// Start launches the background poll loop. It is a no-op when disabled.
func (s *Scheduler) Start() {
	if !s.enabled {
		log.Println("Job Queue scheduler disabled (JOB_QUEUE_ENABLED=false)")
		return
	}
	if err := s.ensureLockTable(); err != nil {
		log.Printf("Job Queue scheduler: failed to create lock table, not starting: %v", err)
		return
	}
	log.Printf("Job Queue scheduler started (interval %s, pod %s)", s.interval, s.podID)
	go s.loop()
}

// Stop signals the loop to exit and waits for it to finish.
func (s *Scheduler) Stop() {
	if !s.enabled {
		return
	}
	close(s.stop)
	select {
	case <-s.done:
	case <-time.After(10 * time.Second):
		log.Println("Job Queue scheduler: stop timed out")
	}
}

func (s *Scheduler) loop() {
	defer close(s.done)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.tick()
		case <-s.stop:
			return
		}
	}
}

// tick runs one scheduling pass: acquire the lock, then process every company.
func (s *Scheduler) tick() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Job Queue scheduler: recovered from panic in tick: %v", r)
		}
	}()

	acquired, err := s.tryAcquireLock()
	if err != nil {
		log.Printf("Job Queue scheduler: lock error: %v", err)
		return
	}
	if !acquired {
		return // another instance owns this tick
	}
	defer s.releaseLock()

	companies, err := s.companies()
	if err != nil {
		log.Printf("Job Queue scheduler: failed to list companies: %v", err)
		return
	}
	for _, company := range companies {
		s.processCompany(company)
	}
}

// processCompany scans one company's Job_Queue for due jobs and runs them.
func (s *Scheduler) processCompany(company string) {
	var jq tables.JobQueue
	jq.InitWithDBType(s.db, company, s.dbType)

	now := types.Now()
	var dueNos []string
	if jq.FindSet() {
		for {
			if jq.Status == gtables.JobQueue_Status.Ready && isDue(jq.Next_start, now) {
				dueNos = append(dueNos, jq.No.String())
			}
			if !jq.Next() {
				break
			}
		}
	}

	// Run after the scan completes so running a job (which may query/modify the
	// same table) can't disturb the FindSet cursor. Re-Get each job fresh.
	for _, no := range dueNos {
		var job tables.JobQueue
		job.InitWithDBType(s.db, company, s.dbType)
		if job.Get(no) {
			s.runJob(company, &job)
		}
	}
}

// isDue reports whether a job with the given next_start is due to run now.
// A zero next_start means "run as soon as possible".
func isDue(nextStart, now types.DateTime) bool {
	return nextStart.IsZero() || !nextStart.After(now)
}

// runJob executes one job's codeunit and applies the outcome (reschedule + notify).
func (s *Scheduler) runJob(company string, job *tables.JobQueue) {
	// Codeunits read the ambient context for company/user; a scheduled run has no
	// user, so we identify it as SCHEDULER (stamped onto Job_Queue_Entry.user_id).
	fcodeunits.SetCurrentContext("SCHEDULER", "SCHEDULER", company, "en-US")
	defer fcodeunits.ClearCurrentContext()

	start := types.Now()
	runErr := s.execute(company, job)

	s.finish(company, job.No.String(), runErr)
	s.notify(job, runErr, start)
}

// execute looks up and runs the codeunit referenced by object_id_to_run.
func (s *Scheduler) execute(company string, job *tables.JobQueue) error {
	factory, ok := codeunits.Get(job.Object_id_to_run)
	if !ok {
		return fmt.Errorf("codeunit %d not found", job.Object_id_to_run)
	}
	cu := factory(s.db, company, s.dbType)

	result, err := cu.Run(job)
	if err != nil {
		return err
	}
	if !result.Success {
		msg := result.Message
		if result.Dialog != nil && result.Dialog.Message != "" {
			msg = result.Dialog.Message
		}
		if msg == "" {
			msg = "job reported failure"
		}
		return errors.New(msg)
	}
	return nil
}

// finish re-reads the job and updates its status / next_start based on the run
// outcome and its recurrence. On error the job is set to Error (stops re-runs
// until a user resets it). A successful Once job is set to On Hold.
func (s *Scheduler) finish(company, jobNo string, runErr error) {
	var job tables.JobQueue
	job.InitWithDBType(s.db, company, s.dbType)
	if !job.Get(jobNo) {
		return
	}

	if runErr != nil {
		job.Status = gtables.JobQueue_Status.Error
		if err := modify(&job); err != nil {
			log.Printf("Job Queue scheduler: failed to set Error status on %s/%s: %v", company, jobNo, err)
		}
		return
	}

	now := types.Now()
	switch job.Recurrence {
	case gtables.JobQueue_Recurrence.Once:
		job.Status = gtables.JobQueue_Status.OnHold // do not reschedule
	case gtables.JobQueue_Recurrence.Minutes:
		mins := job.Minutes_between_run
		if mins < 1 {
			mins = 1
		}
		job.Next_start = now.AddMinutes(mins)
	case gtables.JobQueue_Recurrence.Hourly:
		job.Next_start = now.AddHours(1)
	case gtables.JobQueue_Recurrence.Daily:
		job.Next_start = types.NewDateTimeFromTime(now.Time.AddDate(0, 0, 1))
	case gtables.JobQueue_Recurrence.Weekly:
		job.Next_start = types.NewDateTimeFromTime(now.Time.AddDate(0, 0, 7))
	case gtables.JobQueue_Recurrence.Monthly:
		job.Next_start = types.NewDateTimeFromTime(now.Time.AddDate(0, 1, 0))
	default:
		// Unknown/Once-like: don't reschedule to avoid a busy loop.
		job.Status = gtables.JobQueue_Status.OnHold
	}

	if err := modify(&job); err != nil {
		log.Printf("Job Queue scheduler: failed to reschedule %s/%s: %v", company, jobNo, err)
	}
}

// notify emails the job's notification_email according to notify_on.
func (s *Scheduler) notify(job *tables.JobQueue, runErr error, start types.DateTime) {
	to := strings.TrimSpace(job.Notification_email.String())
	if to == "" {
		return
	}

	send := false
	switch job.Notify_on {
	case gtables.JobQueue_Notify_on.Always:
		send = true
	case gtables.JobQueue_Notify_on.OnError:
		send = runErr != nil
	case gtables.JobQueue_Notify_on.Never:
		send = false
	}
	if !send {
		return
	}

	subject, body := formatNotification(job, runErr, start)
	if err := s.currentMailer().Send(to, subject, body); err != nil {
		log.Printf("Job Queue scheduler: notification email to %s failed: %v", to, err)
	}
}

// formatNotification builds the notification subject and body.
// TODO: localize via i18n once notification recipients carry a language.
func formatNotification(job *tables.JobQueue, runErr error, start types.DateTime) (string, string) {
	outcome := "Success"
	if runErr != nil {
		outcome = "Error"
	}
	desc := job.Description.String()
	subject := fmt.Sprintf("Job Queue %s: %s", outcome, job.No.String())

	var b strings.Builder
	fmt.Fprintf(&b, "Job:         %s\n", job.No.String())
	if desc != "" {
		fmt.Fprintf(&b, "Description: %s\n", desc)
	}
	fmt.Fprintf(&b, "Codeunit:    %d\n", job.Object_id_to_run)
	fmt.Fprintf(&b, "Outcome:     %s\n", outcome)
	fmt.Fprintf(&b, "Started:     %s\n", start.String())
	fmt.Fprintf(&b, "Finished:    %s\n", types.Now().String())
	if runErr != nil {
		fmt.Fprintf(&b, "\nError: %s\n", runErr.Error())
	}
	return subject, b.String()
}

// modify persists changes to a job, returning an error if the write fails.
func modify(job *tables.JobQueue) error {
	if !job.Modify(true) {
		return errors.New("Modify returned false")
	}
	return nil
}
