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

## Feature: Job Queue — Automatic / Scheduled Execution ✅ IMPLEMENTED

**Status:** implemented. In-process DB-polling scheduler (`backend/business-logic/scheduler/`)
runs `Job_Queue` jobs on a `recurrence` schedule and emails notifications
(`backend/foundation/mail/`). Scheduler config: `JOB_QUEUE_ENABLED` (default on),
`JOB_QUEUE_POLL_INTERVAL` (seconds, default 60). **SMTP is configured in the DB** via the global
`SMTP_Setup` table (table 409, Settings → SMTP Setup); no env-var SMTP config. The design notes
below are retained as reference.

`SMTP_Setup` is the first **BC-style setup table** (see the framework feature below).

Not yet done (follow-ups): localize notification email via i18n; per-job leases (currently one global
`_scheduler_lock`); "In Process" start-entry lifecycle (codeunits still self-log their entries);
mask/encrypt the SMTP password (currently stored and rendered in plain text).

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

## Framework: BC-style Setup Tables ✅ IMPLEMENTED

Reusable support for NAV/BC-style singleton **setup tables** — a single record identified by a blank
`Primary Key` (Code[20]). To add one: create a table YAML with a `primary_key` (Code[20], not
required) field and `setup_table: true`, and a list page showing the real fields (omit the PK column
and New/Delete). The framework handles the rest:

- **tablegen**: `setup_table: true` → generates `IsSetupTable()` (on the `Table` interface).
- **Auto-create**: the list endpoint creates the single blank-PK record on first access, so it is
  always present (BC "the setup record is always there").
- **Blank-PK routing**: no-id API routes (`GET /card`, `PUT /modify`, `DELETE /delete`) plus handlers
  that accept an empty id for setup tables; the frontend api client omits the id segment when blank.

Works for global or company-scoped tables. First consumer: `SMTP_Setup` (table 409). Follow-up:
mask/encrypt setup fields flagged sensitive (e.g. the SMTP password) instead of plain text.

---

## Feature: Editable List — BC Record Entry Behavior

**Status:** planned, not started. Inserting a row on an editable list page does not match Business
Central. Reference material: seven BC screenshots in `screenshots/GeneralJournal01-07.png` (General
Journals, batch CBI-RECON) capturing the real lifecycle. The decisive frame is 04 — selecting
`Account No. 01013` makes "✓ Saved" appear **while the cursor is still on the row**, and
simultaneously auto-fills Account Name, Description, Gen. Bus. Posting Group and Gen. Posting Type.
So BC inserts on the **first field the user validates with a value**, not on row-leave, and one
field's validation populates siblings. OpenERP does neither.

### Goal
A new row arrives pre-populated with defaults but uncommitted; it is INSERTed the moment the user
validates the first field with a value, while the cursor is still on the row; validating one field
can populate sibling fields; abandoning an untouched new row discards it silently.

### Observed BC lifecycle (from the screenshots)
- **ArrowDown past the last row** (02) — a new row appears **already populated** (Posting Date, VAT
  Date, Document No., Account Type, Bal. Account Type, Amount 0,00), yet the footer still reads
  "Number of Lines **2**" → the record is not inserted.
- **Lookup on Account No.** (03) — the dropdown opens over the still-uncommitted row.
- **Select 01013** (04) — "✓ Saved" appears; INSERT fires here, on the row. Four sibling fields
  auto-fill from that single validation.
- **Edit Description** (05–07) — every subsequent edit is a MODIFY.
- **ArrowUp out of an untouched new row** — the row is discarded, never inserted.

### The insert lifecycle (the core rule)
Replaces today's row-leave rule end to end.
- **Row created** — populated from the init endpoint, marked `_isNew`, not persisted.
- **INSERT** — fires on the first field the **user** validates with a non-empty value, on the row.
  On success `_isNew` clears and the row holds a real primary key.
- **MODIFY** — every subsequent field commit on that row.
- **DISCARD** — leaving a row on which the user validated nothing removes it client-side, no API call.
- **Critical subtlety:** init-supplied defaults must **not** count toward the insert trigger. In
  screenshot 02 the row already carries Posting Date, Document No., Account Type and Amount while the
  footer still reads 2 lines. Track user-originated edits explicitly (a `_touched` set on the row, or
  a diff against the pristine init payload) — do **not** test "has any non-empty field", which would
  insert the row the instant it is created.
- Keep the existing "only one uncommitted new row at a time" rule; reuse `isEmptyNewRecord`
  (`frontend/src/lib/components/pages/ListPage.svelte:802`) and `cleanupEmptyNewRows` (`:807`).
- Retire the `forceInsert` parameter on `handleCellBlur` (`:690`, gate at `:733`) — insert timing
  stops being a function of *where focus went* and becomes a function of *what the user changed*,
  which also removes the focus-tracking fragility behind several past bugs.

### New-record initialization (backend)
Nothing like this exists today — no `OnNewRecord`, no `InitRecord`, no No. Series anywhere in the
repo. `ListPage.svelte:537-547` and `:843-851` blank every field client-side, so even the four YAML
`default:` values in the repo never reach a new row.
- New `Table` interface method `InitRecord()` (BC/NAV `OnNewRecord`), defaulting to the static-default
  assignment already generated into `InitWithDBType` (`tools/tablegen/main.go:830-847`), overridable
  per table in `backend/business-logic/tables/*.go` alongside the `OnValidate_*` overrides.
- New endpoint `POST /api/tables/:table/init` returning the initialized record as a map — the same
  `table.ToMap()` shape the insert path already returns (`backend/api/handlers/tables.go:565`).
- `ListPage.svelte` calls it instead of the client-side blank-field loops. Must degrade gracefully:
  on failure fall back to today's blank row rather than blocking data entry.
- **Dependency, not in scope:** the `Document No.` in the screenshots (`CBI-G000000…`) comes from a
  **No. Series**, which does not exist here. `InitRecord` is the hook it will plug into — spec No.
  Series as its own item.

### Cross-field auto-fill on validate (backend)
`POST /api/tables/:table/validate` (`backend/api/handlers/tables.go:709-762`) builds a **fresh empty
record** (line 730), so a trigger cannot see the other entered fields, and returns `Success: true`
with **no `Data`** (759-761). The screenshot-04 auto-fill is impossible today.
- Rework it to accept the **full in-progress record** plus `{field, value}`, hydrate the table
  instance from it instead of using a blank one, run `ValidateField` (which tail-calls
  `OnValidate_<Field>()`), and return the **mutated record** via `table.ToMap()` — mirroring what
  insert and modify already do at `:565` / `:650`.
- `api.validateField` (`frontend/src/lib/services/api.ts:153-173`) returns `{valid, error, record?}`.
- `ListPage.svelte` merges the returned record into `editableRecords[rowIndex]`. **Reuse the existing
  post-`await` guard** — `editableRecords[rowIndex]` can be `undefined` after an await because
  `exitToNavigation()` empties the array; this is the exact bug already fixed at the two
  `Object.assign` sites in `handleCellBlur`.
- `OnValidate_*` needs no signature change — it takes no parameters and its receiver is the record
  (`tools/tablegen/main.go:2486-2490`), so mutating siblings is already possible in Go. Only the
  transport discards it today. Five overrides exist repo-wide, all pure validation.
- **Scope expansion to watch:** validate is currently called only for `table_relation` fields lacking
  an advanced lookup (`ListPage.svelte:709`). BC-style auto-fill means calling it on every field
  commit — decide whether to gate it on a per-field or per-table "has an OnValidate override" flag so
  tables that would do nothing with it don't pay a round trip per cell.

### Trailing blank row and row creation
- BC always renders a blank placeholder row after the last record; ArrowDown from the last record
  moves into it and materializes the new row. Today ArrowDown/Enter on the last row calls
  `insertNewRow(true)` (`:961`, `:1058`, `:1177`) — an explicit create rather than a persistent
  placeholder, so the affordance is invisible until the key is pressed.
- Show a **"✓ Saved" status indicator** in the page header reflecting insert/modify completion, as in
  screenshots 01 and 04. Cheap, and it is the only feedback that the row actually committed.

### Files to create / modify (when implemented)
- `backend/api/handlers/tables.go` — rework `ValidateField`; add the init handler.
- `backend/api/server.go` — register the init route (validate is registered at `:120`).
- `backend/foundation/tables/interface.go` — add `InitRecord()` to the interface.
- `tools/tablegen/main.go` — generate the default `InitRecord()` body from YAML `default:` values;
  regenerate `backend/generated/tables/`.
- `frontend/src/lib/services/api.ts` — validate returns a record; add the init call.
- `frontend/src/lib/types/api.ts` — validate response type.
- `frontend/src/lib/components/pages/ListPage.svelte` — insert lifecycle, `_touched` tracking,
  validate-merge, trailing blank row, saved indicator.
- `CLAUDE.md` — **rewrite the "Delayed Insert" ABSOLUTE RULE** to the first-user-filled-field rule,
  and update the "Cell Value Auto-Save" section to match.

### Open questions (resolve at implementation)
- If the INSERT fails server-side validation, does the row stay uncommitted and editable (recommend
  yes, mirroring `modalSaveBlocked`), or revert?
- Should `_touched` reset after a successful insert, or persist for the row's lifetime?
- Does the init endpoint need the current filter context? BC seeds a journal line from its batch —
  likely yes eventually; decide whether to pass filters now or add later.
- Per-field vs per-table gating for the expanded validate calls.

### Verification (when implemented)
- Unit: insert-trigger predicate — table-driven over (init payload, user edits) → insert / no-insert.
  Must cover "pre-populated row with zero user edits does not insert". Mirror the style of
  `frontend/src/lib/utils/__tests__/recordHelpers.test.ts`.
- Unit (Go): `InitRecord()` applies YAML defaults; an overriding table's `OnValidate_*` that mutates
  siblings is reflected in the returned `ToMap()`.
- Integration (SQLite): POST validate with a partially-filled record; assert the response carries the
  sibling fields the trigger set, not just `success: true`.
- Manual: via `scripts/dev.sh` on an editable list — ArrowDown to create a row and confirm it is
  pre-populated *and* absent from the record count; ArrowUp and confirm it vanished with no API call;
  re-create, fill one field, confirm INSERT fires on the row and the saved indicator appears; edit a
  second field and confirm it is a MODIFY, not a second INSERT.
- Regression: composite-PK tables (`User_Member`) — the old rule deliberately let users fill every PK
  part before inserting; the new rule inserts on the first one. Confirm blank-but-optional PK parts
  still work.
- Regression: re-walk the three keyboard tables in CLAUDE.md; the 3-state cell model must be untouched.

### Not in scope
Footer totals (the screenshots' "Number of Lines / Balance / Total Balance"), No. Series, and the
journal objects themselves — no G/L Account table, journal line table, or journal page exists yet
(only a vestigial `journal_batch_name` column on Customer Ledger Entry). This spec is the **generic
engine** a future journal will sit on. Each of those is its own item.

---

## Feature: List Page — Business Central Parity

**Status:** planned, not started. Separate from the record-entry work above — this is the page-level
surface (selection, column headers, action bar), not the insert lifecycle. The 3-state cell model
(navigation / cell-selected / cell-editing) and every existing ABSOLUTE RULE stay intact; this work
adds the BC surface *around* them. Data handling stays client-side for now, so sorting stays
client-side too.

### Goal
Multi-row selection with bulk actions, interactive column headers, and a grouped action bar with a
row context menu.

### Current state (reuse, don't rebuild)
- Sort state already supports direction — `sortField`/`sortDirection`
  (`frontend/src/lib/components/pages/ListPage.svelte:79-80`), `handleSort` (`:1738`), comparison in
  `sortedRecords` (`:163`). Only the *trigger* needs to move to the header.
- Column customization `ItemCustomization {visible, section?, order?}` in
  `frontend/src/lib/utils/customizationStorage.ts` (localStorage `page-customization-*`,
  `column-widths-*`, `row-numbers-*`). `clearPageCustomizations` (`:62`) exists but has no caller.
- `createDragAndDrop` (`frontend/src/lib/utils/dragAndDrop.svelte.ts`) — used today only by
  `CustomizeFieldsModal.svelte`; reuse it for header drag-reorder.
- Server-side filtering already works end to end: `FilterPane` → `onfilter` →
  `PageRenderer.handleFilterChange` (`:414`) → `loadListData` (`:205`). "Filter to this value" must
  use this path, not a second one.
- `frontend/src/lib/stores/confirm.ts` and `toast.ts` for bulk confirmation and failure reporting.
- Grouping precedent to copy: `MenuGroup {Name, Icon, Items}` (`backend/foundation/pages/types.go:80-85`).
- Dropdown keyboard pattern to mirror: `OptionDropdown.svelte` `handleKeydown` (`:95`).

### Phase 1 — Selection & bulk actions
- **Fix first:** `selectedRecord` (`:329`) and `moveDown`/`moveUp`/`moveLast` (`:1684-1706`) index
  `records` instead of `displayRecords` — so the highlighted row and the acted-on record diverge
  whenever search or sort is active. This is a live bug; multi-select is built on top of it.
- Add `selectedKeys: Set<string>` (keys via `getRecordKey`) with `selectedIndex` as the range anchor.
  Never key selection by array index — search and sort reorder `displayRecords`.
- Leftmost checkbox column, opt-in via `multi_select: true` in page YAML (default off, so existing
  pages are unchanged). Header checkbox = select all / none, tri-state when partial.
- Mouse: click selects one; ctrl/cmd-click toggles; shift-click selects the inclusive range in
  `displayRecords` order.
- Keyboard, navigation mode only: `Ctrl+A` select all, `Shift+ArrowUp/Down` extend from anchor,
  `Space` toggle. These must not leak into cell-selected mode.
- **Interaction with the cell model:** multi-select exists only in navigation mode;
  `enterCellSelected` collapses the selection to the single focused row. This keeps every cell-level
  ABSOLUTE RULE untouched.
- Bulk execution: `handleDelete` and `handleRunObject` (`:335`) iterate the selection sequentially —
  one confirm for the whole batch, per-row failures collected and surfaced together rather than
  aborting, one refresh at the end instead of per row.

### Phase 2 — Column header menus
- New `ColumnHeaderMenu.svelte` — a caret menu per `<th>`: Sort Ascending, Sort Descending, Filter to
  this value, Clear filter, Hide column, Freeze pane up to this column.
- Sort drives the existing client-side `sortField`/`sortDirection`. **Note:** the backend never reads
  `sort_order` — only the unused DTO `backend/api/types/api_types.go:26` and `backend/api/README.md:43`
  mention it — so descending sort is impossible server-side today. Client-side keeps it correct.
- Clicking the header caption sorts (toggling direction), replacing the separate sort button at
  `:2042`, which is not a BC affordance.
- "Filter to this value" pushes through the existing `onfilter` path so FilterPane stays the single
  source of filter truth.
- Hide column writes `visible: false` into `columnCustomizations`; the Customize modal restores it.
- Freeze pane: new `frozenColumn` key in `customizationStorage.ts`, rendered `position: sticky;
  left: …` (the table already does `position: sticky; top: 0` on `thead th` at `:2512`).
- Header drag-to-reorder via `createDragAndDrop`, writing the same `order` field the Customize modal
  writes — one persistence format, two editors.

### Phase 3 — Action bar & context menu
Backend `Action` struct (`backend/foundation/pages/types.go:64-72`) — additive, all optional:
`category` (New / Process / Report / Related / Actions), `image` (icon name, replacing the hardcoded
name-based switch at `ListPage.svelte:1879-1887`), `scope` (page / row / selection), `visible`
(`*bool`). Also change `Enabled bool` (`:71`) to `*bool` — its "Default true" comment is currently
unenforceable because omitted and `false` are indistinguishable; match `Editable`/`Visible`/`ModalCard`.

- Promoted actions stay in the toolbar. **Non-promoted actions get rendered** — grouped into category
  dropdowns. Today `:1857` filters to `promoted` only, so a non-promoted action is reachable **only**
  if it happens to carry a keyboard shortcut.
- Right-click opens a row context menu of `scope: row` and `scope: selection` actions plus the
  built-in Edit / Delete / New, dispatching through the same `handleAction` (`:459`) so there is
  exactly one action code path.
- Enablement becomes selection-aware (`scope: row` needs exactly one selected, `scope: selection`
  needs ≥1), replacing the inline IIFE at `:1858`.
- **Also fix:** `run_page` is handled in `CardPage.svelte` (`:307`, `:440`) but not in ListPage — a
  list action with `run_page` silently does nothing, falling through `:515` into
  `PageRenderer.handleListAction` (`:334`), which has no matching case.

### Files to create / modify (when implemented)
- `frontend/src/lib/components/pages/ListPage.svelte` — selection state, header wiring, action bar,
  context menu, the `displayRecords` indexing fix.
- New `frontend/src/lib/components/pages/ColumnHeaderMenu.svelte`, `RowContextMenu.svelte`,
  `ActionGroupMenu.svelte`.
- New `frontend/src/lib/utils/selection.ts` — pure range/toggle/select-all math, kept out of the
  component so it is unit-testable.
- `frontend/src/lib/utils/customizationStorage.ts` — `frozenColumn` key.
- `backend/foundation/pages/types.go` — new `Action` properties; `PageMetadata.multi_select`.
- `frontend/src/lib/types/pages.ts` — mirror the new action and page properties.
- `translations/{en-US,nb-NO}/common.yaml` — captions for the new menu commands (Sort Ascending,
  Filter to this value, Freeze pane, …). Per CLAUDE.md, never hardcode display text in Svelte.
- `CLAUDE.md` — extend "Generic List Page Behaviors" with the selection and action-bar rules.

### Open questions (resolve at implementation)
- Opt-in `multi_select` per page, or on for every list? (Recommend opt-in first, flip the default once
  it has proven itself.)
- Does selection survive a filter change, or reset? (BC resets.)
- Right-clicking a row outside the current selection — select it first? (BC does; recommend matching.)

### Verification (when implemented)
- Unit: `selection.ts` — shift-range across a sorted/filtered `displayRecords`, ctrl-toggle,
  select-all/none, tri-state header, anchor behavior. Mirror
  `frontend/src/lib/utils/__tests__/recordHelpers.test.ts`.
- Unit: action grouping and enablement — category bucketing, `scope` → enabled given 0/1/N selected,
  `visible: false` omitted.
- Integration: note the honest starting point — `ListPage.svelte` is ~92 KB with **zero** direct test
  coverage, and `@testing-library/svelte` ^5.2.0 is installed but unused, so a component test here
  would be the repo's first; budget for establishing the pattern.
- E2E: both existing specs (`frontend/e2e/login.spec.ts`, `navigation.spec.ts`) stop at `/login` and
  never authenticate — a list-page E2E needs a login fixture that does not exist yet.
- Manual: via `scripts/dev.sh` — shift-select a range with a search active and confirm the acted-on
  rows match the highlighted rows (the bug being fixed); bulk delete; sort descending from the header
  menu; reach a non-promoted action from both the Actions menu and the row context menu.

### Deferred to later phases
- **Filter, search & views** — BC filter pane (Shift+F3), server-side search, full BC filter
  expression syntax, saved Views as a tab strip carrying filters + sort + columns. Related finding:
  `backend/foundation/filters/parser.go` is a richer parser (`>`, `<`, `>=`, `<=`, `?`, plus
  `SanitizeFieldName`) that is **dead code** — zero importers — while the weaker per-table generated
  `parseFilterExpression` runs instead, with broken open-ended ranges (`..X`) and an operator-
  precedence bug (`..` is tested before `<>` and `*`).
- **Server-side paging & sorting** — the `page`/`page_size` plumbing shipped in `6f4904a` is complete
  backend-side but has zero frontend callers, and `sort_order` is never read. Until this lands,
  `total` is not a true count and all sorting must stay client-side.
- **Security follow-up (own item, not UI):** filter field names and `sort_by` from the query string
  are interpolated straight into SQL (`backend/api/handlers/tables.go:333`, `:303`) with no
  validation, while the `SanitizeFieldName` helper that would fix it sits unused in the dead
  `foundation/filters` package.
- Totals/footer row, grouping, FactBox pane, export to Excel/CSV, "Show as chart", row-level style
  expressions, expand/collapse rows.

---

_Not tracked here (identified as noise, not real TODOs): i18n `%1`/`%2` substitution logic,
SQL `$N`/`?` parameter builders, Svelte input `placeholder=` attributes, UUID templates in
`toast.ts`, and `vi.stubGlobal` test helpers._
