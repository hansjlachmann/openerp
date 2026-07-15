# TODO

Consolidated backlog for OpenERP. This is the single source of truth for outstanding
work, gathered from sub-READMEs, `docs/migrations.md`, and inline code markers.
`file:line` references are clickable — verify them before starting, as line numbers drift.

Legend: `- [ ]` open · `- [x]` done. Group headings map to areas of the codebase.

---

## Backend — API

From `backend/api/README.md` (formerly "Production TODO" / "Next Steps").

- [x] JWT authentication — **done** (`backend/api/middleware/auth.go`; per-request JWT in HTTP-only cookie)
- [x] Rate limiting — `backend/api/middleware/ratelimit.go`: generous global limiter
      (300/min per IP, skips `/health`) + strict login limiter (10/min per IP) against
      brute-force. In-memory store; needs Redis for multi-instance.
- [ ] Request validation (field constraints) — **partial**. `ValidateField` does type
      checking + calls per-field `OnValidate_<Field>()` triggers (empty stubs), and DB CHECK
      constraints cover numeric min/max at insert. Missing: server-side enforcement of
      **length** (SQLite ignores `TEXT(n)`), **required/not-empty**, and range at validate
      time. Clean fix: generate these checks into the `OnValidate` stubs from YAML metadata
      (tablegen change).
- [ ] HTTPS/TLS support
- [ ] API versioning
- [ ] WebSocket support for live updates
- [x] Pagination for large datasets — **opt-in** server-side pagination. `ListRecords`
      honors `page`/`page_size`: applies LIMIT/OFFSET via `SetPage` on the generated table
      framework and returns the real filtered `total` via `Count()`. Without `page_size` the
      full set is returned (preserving the frontend's client-side search/sort). Empty pages
      return `[]`, not `null`. Frontend adoption of server-side paging is a separate task.
- [ ] Caching layer (Redis)
- [ ] API documentation (Swagger/OpenAPI)

## Backend — Core / Data Layer

- [x] `backend/foundation/database/repository.go` — removed the unused `Repository` type
      (its `Insert/Update/Delete` were no-op stubs and it had no references anywhere).
- [x] `backend/foundation/migrations/helpers.go` — added `RecreateTable` (the SQLite-safe
      create/copy/drop/rename primitive) and made `ChangeColumnType` work on SQLite via it.
      DropColumn/RenameColumn already used native modern-SQLite syntax.
- [x] `backend/business-logic/tables/user.go` — `User.OnDelete` now cascade-deletes the
      user's `User_Preferences` rows (added `GetDBType()` getter to the tablegen template).
- [ ] `backend/business-logic/tables/definitions/custledgerentry.yaml:144` — Posting Groups
      and Dimensions are simplified with no `table_relation`s; add proper relations.

## Backend — Generated Table Scaffold Stubs

These come from the tablegen template (`tools/tablegen/main.go:2751,2762,2803`) and are
stamped into every generated table file. Each carries three stubs:
`// TODO: Add checks for related records (if any)` (in `OnDelete`),
`// TODO: Update related records if needed`, and
`// TODO: Add your custom business logic methods here`.

Fill in only where real per-table logic is actually needed — most are intentional boilerplate.

- [ ] `backend/business-logic/tables/customer.go` (46, 52, 125)
- [ ] `backend/business-logic/tables/customerledgerentry.go` (45, 51, 115)
- [ ] `backend/business-logic/tables/paymentterms.go` (44, 50, 77)
- [ ] `backend/business-logic/tables/jobqueue.go` (44, 50, 80)
- [ ] `backend/business-logic/tables/jobqueueentry.go` (44, 50, 80)
- [ ] `backend/business-logic/tables/menu.go` (44, 50, 80)
- [ ] `backend/business-logic/tables/language.go` (44, 50, 124)
- [ ] `backend/business-logic/tables/user.go` (58)
- [ ] `backend/business-logic/tables/userpreferences.go` (48, 65)

## Frontend

From `frontend/README.md` (formerly "Next Steps") and inline markers.

- [ ] `frontend/src/lib/components/pages/PageRenderer.svelte:500` — only `List` and `Card`
      page types render; all other types hit the "not yet supported" fallback. Add support
      for additional page types as they are introduced.
- [ ] WebSocket support for real-time updates (pairs with the backend WebSocket item).
- [x] Create Go API backend — done
- [x] YAML page definitions (dynamic page generation) — done
- [x] Generic `PageRenderer` (List/Card) — done

## Migrations

From `docs/migrations.md`.

- [x] SQLite-safe drop-column / alter-column migration helper — done via `RecreateTable`
      + SQLite `ChangeColumnType` in `backend/foundation/migrations/helpers.go`.

---

## Planned Feature: Job Queue — Automatic / Scheduled Execution

**Status:** specified, not implemented. Decisions: recurrence modeled as a new Option field;
execution via an in-process DB-polling scheduler (no cron dependency, no OS cron).

### Goal
Run `Job_Queue` records automatically on a recurring schedule (Minutes / Hourly / Daily / Weekly /
Monthly) or a single time (Once), without a user clicking Run (F9). Multi-company aware and safe to
run across multiple backend instances (Kubernetes).

### Current state (reuse, don't rebuild)
- `Job_Queue` (table 472, `backend/business-logic/tables/definitions/jobqueue.yaml`) already has
  dormant scheduling fields — `status` [On Hold/Ready/Error], `object_id_to_run`, `parameter`,
  `next_start`, `minutes_between_run`, `recurring_job` — **none read by any code today**.
- `Job_Queue_Entry` (table 473): per-run history log (status Success/In Process/Error, user_id,
  description, error_message, start/end times).
- Run helpers: `backend/business-logic/codeunits/jobqueue_helpers.go` — `CreateJobQueueEntry`,
  `SetJobQueueStatus`.
- Codeunit invocation: `codeunits.Get(object_id_to_run)` → `Run(record)` (the path `jobs.go`
  `StartJob` uses, minus HTTP/SSE).
- Per-goroutine context: `fcodeunits.SetCurrentContext/ClearCurrentContext` — must be set before a
  run so `CurrentCompany()` and `CreateJobQueueEntry`'s `user_id` resolve.
- Distributed-lock pattern to copy: `_migration_lock` CAS+TTL in
  `backend/foundation/migrations/runner.go` / `schema.go`.
- Company enumeration: `companyMgr.ListCompanies()` (`backend/foundation/company/company.go`).
- Ticker-loop template: `backend/api/middleware/sessioncache.go` (`time.NewTicker` + `stop chan`).
- Lifecycle plug-in point: `cmd/api-server/main.go` — start scheduler after the server goroutine,
  stop it on `<-quit` alongside `server.Shutdown()`.

### Schema change: recurrence field
Add to `jobqueue.yaml` (then regenerate via `tools/tablegen/main.go`):
- `recurrence` — Option `["Once", "Minutes", "Hourly", "Daily", "Weekly", "Monthly"]`.
- Keep `minutes_between_run` (used when `recurrence == Minutes`); derive "recurring" from
  `recurrence != Once` (leave `recurring_job` for backward compat / UI).
- `notification_email` — Code/Text: recipient address for post-run notifications.
- `notify_on` — Option `["Always", "On Error", "Never"]` (default `Never`): when to email after a run.
- Future (out of v1 scope): `starting_time`, run-on-weekday / day-of-month for BC-style windows.

`next_start` recomputation after a successful run:
- `Once` → do not reschedule; set `status = On Hold`.
- `Minutes` → `now + minutes_between_run`. `Hourly` → `+1h`.
- `Daily/Weekly/Monthly` → `time.AddDate(0,0,1)` / `AddDate(0,0,7)` / `AddDate(0,1,0)` (calendar-correct
  month lengths).

### Execution engine (in-process scheduler)
New package (e.g. `backend/business-logic/scheduler`):
- `Scheduler{db, dbType, companies func()([]string,error), stop chan struct{}}`, ticker loop modeled
  on `sessioncache.go`. Poll interval + enable flag via env (`JOB_QUEUE_POLL_INTERVAL` default ~60s,
  `JOB_QUEUE_ENABLED` default true).
- **Multi-pod safety:** a `_scheduler_lock` table (mirror `_migration_lock`): CAS
  `UPDATE ... SET locked_by=$pod, expires_at=now+TTL WHERE id=1 AND (locked_by IS NULL OR expires_at < now)`.
  Only the winning pod runs the tick; TTL heartbeat frees a dead pod. (Per-job leases are a future
  refinement; one global lock is fine for v1.)
- **Tick body:** acquire lock → for each `company` in `ListCompanies()` → scan `company$Job_Queue`
  for `status == Ready AND next_start <= now` (or null) → for each due job:
  `SetCurrentContext("", "SCHEDULER", company, "en-US")` → `codeunits.Get(object_id_to_run)` →
  build `Job_Queue` record → `Run(record)` → write outcome → `ClearCurrentContext()`.

### Status / run lifecycle
- Write a `Job_Queue_Entry` at **start** with status `In Process` (today entries are written only at
  end — add a start/finish lifecycle helper alongside `CreateJobQueueEntry`).
- Success → update entry to `Success` + end time; recompute `next_start` (or `status = On Hold` if
  `Once`).
- Error → entry `Error` + `error_message`; set `Job_Queue.status = Error` (stops re-runs until a user
  sets it back to Ready) via `SetJobQueueStatus`.
- Overlap guard: skip a job whose latest entry is still `In Process` (or use a per-job lease).

### Email notification (post-run)
After a scheduled run finishes, optionally email a notification to `notification_email`, gated by
`notify_on`:
- `Always` → email on both success and error.
- `On Error` → email only when the run failed.
- `Never` (default) → no email.
Skip entirely when `notification_email` is blank. The email body should include job `no` /
description, outcome (Success/Error), start/end time, and `error_message` on failure — sourced from
the `Job_Queue_Entry` just written, so notification happens right after the entry is finalized in the
scheduler tick (not inside the codeunit).

**No email infrastructure exists yet** — this is a new component:
- New mailer package (e.g. `backend/foundation/mail`) using stdlib `net/smtp` (no new dependency) or a
  thin wrapper; send is best-effort and must not fail/block the job run (log on failure).
- SMTP config via env: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`,
  and an enable flag (`SMTP_ENABLED`, default false so dev/tests don't attempt delivery).
- User-facing strings (subject/body) via `i18n` per the project's no-hardcoded-strings rule.

### Non-interactive constraint (important)
Scheduled runs have no user, so codeunits calling `Confirm` / `RequestInput` can't be driven
normally. v1: run with a **headless dialog** that auto-declines confirms and returns empty for input,
and mark interactive codeunits as not schedulable. `helloworld.go` and `nav_report_runner.go` (report
generation — uses progress but no user confirms) are the realistic scheduled targets.

### Files to create / modify (when implemented)
- `backend/business-logic/tables/definitions/jobqueue.yaml` — add `recurrence`, `notification_email`,
  `notify_on`; regenerate.
- New `backend/business-logic/scheduler/scheduler.go` — ticker loop + lock + per-company scan + run
  + post-run email dispatch.
- New `backend/foundation/mail/` — SMTP mailer (env-configured, best-effort).
- Lock table DDL — migration or scheduler-owned `CREATE TABLE IF NOT EXISTS _scheduler_lock`.
- `backend/business-logic/codeunits/jobqueue_helpers.go` — start/finish entry lifecycle helper.
- `cmd/api-server/main.go` — start/stop the scheduler in the lifecycle.
- Page 672 YAML — surface `recurrence`, `notification_email`, `notify_on` for editing (renders
  generically).
- `backend/foundation/codeunits/dialog.go` — headless dialog for non-interactive runs.

### Open questions (resolve at implementation)
- Sequential vs bounded-worker execution per tick (start sequential).
- Timezone for Daily/Weekly/Monthly boundaries (UTC vs server local) — pick and document.
- Missed runs while down: single catch-up vs skip (recommend single catch-up).

### Verification (when implemented)
- Unit: recurrence → `next_start` math (table-driven, incl. month-length), lock CAS acquire/release,
  Once → On Hold.
- Integration (SQLite): seed `Job_Queue` `status=Ready, object_id_to_run=<helloworld>,
  recurrence=Minutes, minutes_between_run=1, next_start=past`; run one tick; assert a `Success`
  `Job_Queue_Entry` was written and `next_start` advanced.
- Multi-pod: two schedulers, one DB; assert a due job runs once per tick.
- Email: with a stub/mock SMTP sender, assert `notify_on` gating (Always vs On Error vs Never) and
  that a blank `notification_email` sends nothing; assert a failed SMTP send doesn't break the run.
- Manual: via `scripts/dev.sh`, set a job Ready with near `next_start`, watch it run and log on page 673.

---

_Not tracked here (identified as noise, not real TODOs): i18n `%1`/`%2` substitution logic,
SQL `$N`/`?` parameter builders, Svelte input `placeholder=` attributes, UUID templates in
`toast.ts`, and `vi.stubGlobal` test helpers._
