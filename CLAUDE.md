# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

OpenERP is a full-stack ERP system inspired by classic ERP systems. Go backend (Fiber) + SvelteKit frontend + PostgreSQL/SQLite.

**Required runtimes**: Go 1.24, Node.js 22. Always target these versions.

## Business Logic (ABSOLUTE RULE)

**Business logic is always coded in the backend** — in the YAML definitions (tables, fields, relations, FlowFields, keys, pages' sources) or in the Go code (table triggers `OnInsert`/`OnModify`/`OnDelete`/`OnRename`/`OnValidate_*`, codeunits, `backend/business-logic/…`). Business rules, validations, calculations, defaults, posting and delete rules never live in the frontend: the frontend only shows data, collects input and calls the API, and the backend enforces every rule (also for API calls that do not come from the UI).
- Definition: a change that does not affect the backend is **not** business logic by definition (layout, navigation, keyboard handling, styling, client-side convenience). A change that adds or changes a business rule must be made in the backend YAML or Go code.
- Business logic work is collected in `TODO.md` under "Business Logic (later)".

## Public Repository (ABSOLUTE RULE)

This repository is **public** on GitHub, and so are its issues, pull requests, release notes and the container images on ghcr.io. Everything committed or pushed is readable by anyone, and stays in the git history even after it is deleted from the files.
- **Never** put confidential or installation-specific details into tracked files, commit messages, PR/issue text or release notes: server hostnames, domains, IP addresses (also private ones like `10.x` / `192.168.x`), ports of internal services, user names on servers, paths on servers, customer or company names of installations, certificate names/issuers/expiry, passwords, tokens, API keys, SMTP credentials, private keys and certificates (`*.pfx`, `*.p12`, `*.key`, `*.pem`, `certs/`), database dumps, backups and real customer data.
- Site settings come from the environment: untracked `prod.env` (template `prod.env.example` with placeholders only), Kubernetes secrets, or tables such as SMTP Setup. Code defaults are generic (`localhost`), never a real address — e.g. the NAV proxy is `NAV_PROXY_URL`, default `http://localhost:5009`.
- Describe production work generically in `TODO.md` and docs ("Production: upgraded to 0.1.84"), never with its host, domain or address. Examples use `example.com` / `x.no` style placeholders.
- Do not commit build outputs or local data: binaries (`api-server`), SQLite databases (`*.db`, `-wal`, `-shm`), `.env` files other than the CI-managed `.env` (only `APP_VERSION`).
- **Neutral wording (ABSOLUTE RULE)**: never name the commercial ERP products whose conventions this project follows, their vendor, or their programming languages — in commit messages, PR titles and texts, release notes, CHANGELOG entries, code comments, docs, TODO.md, test names or UI texts. Describe the behaviour itself ("field numbers identify fields", "date formula", "classic ERP convention"). Local reference material (screenshots, other products' sources) stays out of the repository (`.git/info/exclude`) and is never referred to. The only exception is an integration with a real external system, named as it is (the report proxy codeunit, `NAV_PROXY_URL`).
- **Before every commit and push**, check the diff and the commit message for such details (e.g. `git diff --cached | grep -nE '([0-9]{1,3}\.){3}[0-9]{1,3}|BEGIN .*PRIVATE KEY|password\s*[:=]'`). If something slipped into a pushed commit, deleting it in a new commit is not enough — tell the user: it needs a history rewrite (`git filter-repo`), a force-push and a GitHub Support purge of cached commits and PR refs.

## Build & Run Commands

### Backend (Go 1.24)
```bash
go run ./cmd/api-server              # Run (interactive: prompts for DB path & company)
go build -o api-server ./cmd/api-server  # Build binary
go fmt ./...                         # Format
go vet ./...                         # Lint
go test -v -race ./...               # Test with race detection
```

### Frontend (SvelteKit + Svelte 5)
```bash
cd frontend
npm install                          # Install deps
npm run dev                          # Dev server on :5173 (proxies /api to :8080)
npm run build                        # Production build
npm run check                        # TypeScript/Svelte type checking
npm run test                         # Unit tests (Vitest)
npm run test:e2e                     # E2E tests (Playwright)
```

### Code Generation
```bash
go generate ./backend/generated/tables   # after changing a table YAML in backend/definitions/tables
```
Commit the regenerated `backend/generated/tables`; CI fails when it is not up to date.

### Docker
```bash
docker-compose up                    # Full stack (backend + frontend + postgres + nginx)
scripts/prod.sh pull && scripts/prod.sh up -d  # Production (settings/secrets in untracked prod.env)
scripts/backup.sh                    # Production database backup (pg_dump, prunes old automatic dumps)
```

Production operations (upgrade, backup/restore, reboot, ports, passwords): `docs/operations.md`. Server settings and secrets live in the untracked `prod.env` (template `prod.env.example`), never in the tracked `.env` (CI rewrites it with `APP_VERSION`). All prod services have `restart: unless-stopped`; only nginx (80/443) is published to the network — db on `127.0.0.1`, backend/frontend on `PUBLISH_HOST` (default `127.0.0.1`). Never put server addresses or other site details into the repository.

## Architecture

```
cmd/api-server/          Entry point - DB init, object registry setup, server start
backend/
  api/                   Fiber HTTP server, routes, handlers, middleware
  definitions/           Metadata (YAML): what extensions extend
    tables/              Table definitions → tablegen
    pages/               Page definitions (YAML-driven UI metadata, read at runtime)
    menus/               Menu definitions (read at runtime)
  generated/tables/      Generated from definitions/tables by tablegen — never edit
  business-logic/        Maintained business logic (triggers, codeunits), no boilerplate
    tables/              Table triggers (one logic type per table) and table helpers
    codeunits/           Business logic units (runnable via API)
    migrations/          Versioned schema migrations (see docs/migrations.md)
  foundation/            Core framework
    database/            DB abstraction (SQLite dev, PostgreSQL prod - auto-switches on DB_HOST)
    session/             Per-request session context (user, company, language)
    objects/             Object registry for tables/pages/codeunits
    company/             Multi-company management (tables prefixed: "company$TableName")
    i18n/                Internationalization (en-US, nb-NO)
    types/               Core type system (Code, Text, Integer, Decimal, etc.)
    filters/             Filter expression parser for list queries
    migrations/          Migration infrastructure (locking, runner, helpers)
frontend/
  src/routes/            SvelteKit pages (pages/[id]/, pages/[id]/[recordid]/)
  src/lib/components/    Reusable Svelte components
  src/lib/services/      API client
  src/lib/stores/        Svelte stores (session, toast, confirm, breadcrumbs)
  src/lib/utils/         Utilities including keyboard shortcuts
tools/
  tablegen/              Generates Go code from YAML table definitions
  extmerge/              Merges extension definitions with core at build time
translations/            i18n JSON files
```

### Frontend Path Aliases (svelte.config.js)
- `$components` → `src/lib/components`
- `$stores` → `src/lib/stores`
- `$services` → `src/lib/services`
- `$types` → `src/lib/types`
- `$utils` → `src/lib/utils`

## Key Patterns

- **Object Registry**: Tables, pages, and codeunits are registered at startup in `cmd/api-server/main.go`. The registry enables dynamic, type-safe access throughout the system.
- **Table business logic (ABSOLUTE RULE: generated code is never edited)**: three folders, one rule each.
  - `backend/definitions/tables/*.yaml` is the table; `go generate ./backend/generated/tables` writes `backend/generated/tables/<table>_gen.go` (record type `gtables.Customer` with fields, CRUD, trigger dispatch, `CheckFields`) and `tables_gen.go` (table registry `GetTableFactory`/`ListTableNames`, `NewRecords` for the object registry, `EncryptStoredSecrets`). A new table needs no registration anywhere. The registry name is the YAML `name`, or `api_name` where the API/pages use another (`Payment_terms`, `Customer_ledger_entry`, `User_preferences`).
  - `backend/business-logic/tables/<table>.go` holds only the business logic: a logic type registered in `init` with `gtables.Register<Table>Logic(…)`, whose methods are the triggers it needs — `OnInsert`, `OnModify`, `OnDelete`, `OnRename` (a primary key field changed; `rec.OldValue(field)` is xRec) and `OnValidate_<Field>`, each `func (customerLogic) OnValidate_Name(rec *gtables.Customer) error`. Add a trigger when it is needed; the available ones are listed at the top of the generated file. Tablegen creates this file once for a new table (empty OnInsert/OnModify/OnDelete/OnRename) and never overwrites it. Helpers that are not triggers are plain functions (`tables.SetPassword(user, pw)`, `tables.UserThemes(…)`).
  - **Field numbers (ABSOLUTE RULE, field IDs)**: every field in a table YAML (also FlowFields and FlowFilters) has `no:`, unique in the table — tablegen refuses a missing or duplicate number. The base application uses its field series (1–999), extension fields their app's series (extmerge refuses others and numbers or names already used, and marks them with `app:`). Tables with the classic layout use its numbers (Customer 1 = No., 2 = Name, 55 = Date Filter …) and 900–999 for their other fields; other tables number in YAML order. A number is the field's identity: never change it and never reuse the number of a removed field (renaming the field keeps the number). Generated: `gtables.Customer_FieldNo.Name`, `rec.FieldNo("name")`, `rec.FieldName(2)`, `FieldInfo.No`.
  - The field rules of the definition (`required`, `length`, `validation`) are generated (`CheckFields`) and run after OnInsert/OnModify (with `runTrigger`) — never repeat them in a trigger.
  - Every registered logic of a table runs, in registration order (base application first, later extensions); the first error stops the operation. A method named `On…` that is not a trigger of the table (misspelled, or the field was renamed/removed in the YAML) or has the wrong signature panics at startup with its name. Tests replace the logic of one record with `rec.SetLogic(…)`.
  - The logic is registered by importing `backend/business-logic/tables`: `cmd/api-server` does, and so do the packages that write records (`_` import in codeunits, demodata, scheduler, jobqueue) so their tests run the triggers too.
- **System functions (classic ERP functions in Go)**: business logic (table triggers, codeunits) uses the classic functions with their classic names, so it reads like classic ERP code — never hand-rolled equivalents:
  - `backend/foundation/system`: text `StrSubstNo` (`%1`…`%9`, `#1###`), `CopyStr`, `DelChr` (`=`, `<`, `>`), `StrLen`, `StrPos`, `PadStr`, `IncStr`, `SelectStr`, `ConvertStr`, `InsStr`, `DelStr`, `StrCheckSum`, `UpperCase`, `LowerCase`; dates `CalcDate` (date formulas `<1M+CM>`, `-CM`, `WD3` …), `DMY2Date`, `Date2DMY`, `DWY2Date`, `Date2DWY` (ISO weeks), `Today`, `WorkDate` (today for now), `Time`, `CurrentDateTime`, `CreateDateTime`, `DT2Date`, `DT2Time`; numbers `Round(x, 0.01, "=" | "<" | ">")`, `Abs`, `Power`, `Random`; `CreateGuid`, `IsNullGuid`, `UserId`; `Format` / `Evaluate` (Go-native, language-independent formats: dates ISO, decimals `1234.5`, `true`/`false` — the frontend formats for the user); `CompressArray` (arrays: Go's `len` for the length); `Message`, `Error`, `Confirm`, `StrMenu` (options as a comma-separated string → the chosen number, 0 = cancelled), `Hyperlink` (http/https/mailto, opens a new tab) and `OpenWindow` (the progress window: template `#1###` text fields and `@2@@@` bars 0–10000, `Update`, `Close`) for codeunits (pass translated texts). Records: `CopyFilter(field, &other, otherField)`, `SetRange(field)` without values removes the filter, every table has its number as `gtables.<Table>TableID`; `Find("-" | "+" | "=" | ">" | "<" | "=>" …)` in the current key and direction (`>`/`<`: the record after/before the current one — a row-value comparison, not for keys with FlowFields); `FilterGroup(n)` (filters in layers: SetRange/SetFilter/GetFilter work on the current group, all groups apply; code puts the filters the user must keep in group 2 and goes back to 0; `Reset` clears all groups); `SetSelectionFilter()` in a codeunit run from a list (the API passes the list's selection — keys, or all of the list's query, at most 1000 — and the record limits itself to them; with no multi-selection the current record). FlowField filters: `upper_limit: true` on a `type: filter` flow filter applies only the upper end of the FlowFilter (balance up to a date, `flowfilter.UpperLimit`). Temporary records: `rec.InitTemporary(db, company, dbType)` — the record's own table lives in a private in-memory SQLite database (Insert/Modify/Delete/Find/filters/keys/CalcSums as usual, FindSet reads all at once), nothing reaches the real database; triggers and FlowFields still use the real one, a rename does not cascade. `rec.Consistent(false)` marks the table inconsistent in its transaction; commit such transactions with `ftables.Commit(tx)` (refuses and rolls back while a table is inconsistent, `ERR_INCONSISTENT`). Closing dates: `system.ClosingDate`/`NormalDate` (`types.Date` at 23:59:59, text `C2026-12-31`, sorts after the date); they cannot be stored in date fields yet. Pages from code: a page variable `p := system.NewPage(22); p.SetTableView(&rec); p.SetRecord(&rec); p.LookupMode(true); if p.RunModal() == system.ActionLookupOK { p.GetRecord(&rec) }` opens a list in a modal over the running codeunit (filters of all filter groups, positioned on the record; read-only). Lookup mode shows OK and Cancel: OK or Enter/double-click → `ActionLookupOK`, Cancel/close/Esc → `ActionLookupCancel`; without lookup mode closing returns `ActionOK`. `system.RunModal(pageID, &rec, lookupMode)` does it in one call. Async codeunits only (without a dialog: a cancel). Action buttons: every page action (list pages, cards, subpages, menu buttons) is an `ActionButton` — one size (32px), no border, an icon and the caption cut off with "…" (tooltip: the whole caption). The icon is the action's `image:` in the page YAML, a name of the icon set in `frontend/src/lib/components/icons/actionIcons.ts` (New, Edit, Delete, Copy, Refresh, Card, Lines, Entries, Document, Customer, Run, Post, Approve, Cancel, Release, Reopen, Calculate, Undo, Send, Email, Print, Statistics, Export, Import, Calendar, History, Attach, Back, Navigate, Filter, Search, Customize, Rows, Setup, Info, Warning; case-insensitive, a few aliases); without `image:` a bullet point. Add new icons to that set (a vitest checks every YAML image is in it). Page functions and menus: a page action with `items:` is a menu button (dropdown) whose items are actions of their own (`run_object`, `run_page`, `shortcut`). `run_object: <type>:<object>.<function>` (e.g. `table:Job_Queue_Entry.ShowLineNo`, `codeunit:900.PostLine`) calls a function with the page's current record; the function must be registered in the business logic with `functions.Register("table"|"codeunit", <table name | codeunit no.>, <name>, functions.For(fn))` (`functions.OnTable(table)` for codeunit functions, `functions.WithDialog()` to ask/show progress) — only registered functions can be called; the API reads the record from the database by its key; the function returns a `codeunits.Result` (e.g. `system.Message`). A test checks that every function a page uses is registered and takes the page's record. Card subpages: a card section with `subpage: {page_id: <list page>, link: {<subpage field>: <card field>}}` shows that list inside the card with the lines of the current record — the list API's `fixed_filters` (exact values in filter group 2, so the user's filters narrow them and cannot replace them), new and pasted lines get the link values, the user filters/searches as on any list; F6/Shift+F6 and Ctrl+↓/↑ move between the card and its subpages (Ctrl+Shift+↓/↑ = previous/next record on every card), the subpage's shortcuts only act while the cursor is in it (`data-own-keys`). Lines of a record that is not saved yet are not shown. `auto_split_key: true` on the lines' list page: a new line below the cursor row gets the line number (last primary key field) halfway to the next line, 10000 steps at the end. Processing-only reports: `business-logic/reports` — one file per report, `Register(id, name, func() Report { return &x{} })` (a new instance per run, state in its fields); `Definition()` returns request page `Options` (shown below the filters under `OptionsCaption` — a text key of the report, default "Options") and `DataItems` (Table, Key/Descending, TableView = fixed filters in group 2, RequestFilterFields = the filters the request page starts with, Link = child field → parent field in group 4 like `DataItemLink=Customer No.=FIELD(No.)`, nested DataItems = indented; several at one level run one after another) and triggers wrapped with `reports.Trigger(func(rec *gtables.X, run *reports.Run) error)`; return `reports.Skip()`, `Break()`, `Quit()`; read options with `run.Option*`, texts with `run.Text(key)` from `translations/<lang>/reports.yaml` (`reports.<id>.caption`, `.options.<name>`, `.texts.<key>`), finish with `run.Message`. Run from code with `reports.Execute(id, db, company, dbType, reports.RunOptions{…})`; pages use `run_object: report:<id>` (job API `report_id`). The request page shows the options and, per data item, filters on any field of its table — the list page's filter pane behaviour, applied on the server with `ftables.ApplyFilter` (the same function as the list's filters: stored queryable fields, Sum/Count FlowFields). Ask the user only when `system.GuiAllowed()`. Page triggers: `business-logic/pages/<page>.go` registers `pages.OnAfterGetRecord(pageID, func(rec *gtables.X, page *pages.Control) { page.SetEditable(field, ok); page.SetVisible(field, ok) })`; cards ask `POST /api/pages/:id/field-states` for the record they show (only pages with `has_record_triggers`).
  - Generated record methods: `TestField`, `FieldError` (translated `apperrors`), `CalcSums`, `LockTable` (`SELECT … FOR UPDATE` on Postgres, inside a transaction), `Init` (keeps the key), `Validate`, `TableCaption`/`FieldCaption`, `CompanyName`, `ChangeCompany`, `TransferFields` (by field number and type), `Copy`, `Rename(newKey…)`, `Find("-" | "+" | "=")`, `GetFilter`, `GetRangeMin`/`GetRangeMax`, `SetRecFilter`, `Mark`/`MarkedOnly`/`ClearMarks`, `RecordId`, `CountApprox`, `MaxStrLen`, `FieldNo`/`FieldName`, `FieldValue`/`SetFieldValue`.
  - `codeunits.Run(id, rec)` runs a codeunit on a record of its source table.
  - The easiest Go way where a classic function does not fit Go: 1-based character positions, an error result where the function can fail (`CalcDate`, `DMY2Date`), `Evaluate` returns ok. Functions left out on purpose (Go has its own way, or they need more first) are listed in `TODO.md`.
- **Company-scoped tables**: All business data tables are prefixed with the company name (e.g., `"cronus$Customer"`). System tables (User, Language) are global.
- **Database auto-detection**: If `DB_HOST` env var is set, uses PostgreSQL; otherwise SQLite with interactive prompts.
- **Opening a table in code**: always `InitWithDBType(db, company, dbType)` with the session's/handler's database type — there is deliberately no `Init(db, company)`: the type decides the SQL placeholders, and the old SQLite default silently broke code on PostgreSQL (production).
- **Table sync**: Additive schema changes (new tables/columns) happen automatically on startup. Destructive changes require migrations.
- **Migrations**: Sequential versioned files in `backend/business-logic/migrations/`. Auto-registered via `init()`. Use `ctx.ForEachCompanyTable()` to apply across all companies. Supports distributed locking for Kubernetes.
- **Object and field numbers, `application.yaml` (ABSOLUTE RULE)**: every app has an `application.yaml` — the base application's is `backend/definitions/application.yaml`: a GUID `id` that never changes, name, publisher, contact (public repository), `version` (release-please updates it), `number_series` (`objects`, `fields`) and a separate `system_tables` section. The base application uses objects 1–999 (tables, pages, codeunits, reports — unique per object type), fields 1–999 and system tables 2000000000–2000000199. **Use the classic numbers** where the object follows a classic one: Payment Terms 3, Language 8, Country/Region 9, Customer 18, Cust. Ledger Entry 21, SMTP Setup 409, Job Queue 472, Job Queue Log 474; system tables User 2000000002, Member Of 2000000003, User Role 2000000004, Permission 2000000005, Company 2000000006, User Personalization 2000000073, User Metadata 2000000075, Installed App 2000000153; pages Payment Terms 4, Languages 9, Countries 10, Customer Card 21, Customer List 22, Customer Ledger Entries 25, Users 59, Companies 357. Other objects use 900–999: Menu 900, Theme 901–903, pages 900–907, codeunits 900–903 (990/991 demos). Checks: tablegen (table and field numbers), extmerge (an extension's own `application.yaml`, series must not overlap) and startup (`business-logic/apps.CheckObjectNumbers`: every table, page and codeunit in an installed app's series — the server refuses to start otherwise); the table Installed_App (page 907) lists the installed apps, written at startup. Renumbering a stored object number (codeunits in `Job_Queue.object_id_to_run`, pages in `User_Preferences.page_id`) needs a migration (see 010).
- **Field numbers in the database**: the table sync records each physical table's field numbers and columns (`_field_definition`). A field renamed in the YAML with the same number gets its column renamed (data kept), instead of a new empty column; a field whose number changed stops the sync (the server does not start). Never change a field's number; to rename a field, keep its number.
- **Extensions**: Extensions are `.extend.yaml` files merged into `backend/definitions` at build time via `extmerge`, after which the Dockerfile regenerates the table code (extension fields become fields of `gtables.<Table>`); an extension's Go code registers its own logic per table (`EXTENSIONS.md`, "Extension apps"). Keep the base app extensible: never let base code depend on a fixed field list or on being the only registered logic.
- **YAML-driven definitions**: All table definitions, list pages, and card pages are defined in backend YAML files. Never hardcode table structures or page layouts in Go or Svelte code.
- **Generic pages**: All frontend pages (list and card) must be fully generic — driven by page metadata from the backend, not hardcoded per table. Never create table-specific page components.
- **Menu assignment**: The main menu is determined per user via `User.menu` → `Menu.code` (FK). The Menu table defines a `filename` pointing to a YAML menu definition file in `backend/definitions/menus/`. The `filename` must include the `.yaml` extension (e.g., `"jobqueue.yaml"`). Main menu items must always be defined in backend YAML files — never hardcode menu items in the frontend.
- **Frontend proxy**: Vite dev server proxies `/api` to the Go backend at `localhost:8080`.
- **Per-request JWT sessions**: Authentication uses JWT tokens in HTTP-only cookies (`openerp_session`). The auth middleware (`backend/api/middleware/auth.go`) parses the JWT on every request, loads/creates a session, and stores it in `c.Locals("session")`. Handlers access the session via `getSession(c)` (in `handlers/helpers.go`), never via the old `session.GetCurrent()` global. Sessions are cached in-memory (`SessionCache`) keyed by `userID:company`. Login/logout/company-switch re-issue the JWT and update the cache. The `JWT_SECRET` env var must be set in production; in dev a random key is generated (sessions won't survive restarts). `SecureCookie` is automatically enabled when `DB_HOST` is set (production). The `MenuBar` component is mounted once in the root layout and does NOT re-mount on client-side navigation — use `$effect` watching stores to reload data after login/logout.
- **Migration ordering**: Migrations run BEFORE table sync in `EnterCompany`. If a migration needs to INSERT into a table, it must first ensure the table exists with `CREATE TABLE IF NOT EXISTS`. Use idempotent inserts (`ON CONFLICT DO NOTHING` for PostgreSQL, `INSERT OR IGNORE` for SQLite).
- **Case-sensitive DB comparisons**: PostgreSQL string comparison is case-sensitive. Always use the canonical (uppercase) user ID from the database (e.g., `user.User_id.String()`) for permission and lookup queries — never use raw user input directly.
- **Composite primary keys**: Tables can have composite primary keys (e.g., `User_Member` has `user_id + role_id + company`). `TableMetadata` stores `map[string][]string` for all PK fields. The API uses comma-separated PK values in URLs (e.g., `/delete/HANS2,READER,TEST-COMPANY`), parsed by `parseRecordKey()` in the backend.
- **Field type metadata**: The backend sends `field_types` (a `map[field_name]type_string`) in API captions for both page definitions and table record responses. The page endpoint (`/api/pages/:id`) reads types from YAML table definitions via `TableMetadata.GetFieldType()` (returns the YAML types as written: `"bool"`, `"int"`, `"Option"`, `"types.Code"`, `"types.Text"`, `"types.Date"`, …). Test types with the helpers in `utils/fieldHelpers.ts` (`isCodeType`, `isDateType`, `isDateTimeType`), which accept both spellings — comparing with `'code'` matched nothing and left list Code cells lowercase. The table endpoints (`/api/tables/...`) use the Go `FieldType` constants (returns `"Boolean"`, `"Code"`, `"Text"`). The frontend uses the page endpoint's `field_types` for rendering decisions (e.g., boolean checkbox detection).
- **SIFT (Sum Index Fields)**: a table key can declare `sum_index_fields` in its YAML (e.g. Customer Ledger Entry key `customer_open: [customer_no, open]` sums `remaining_amt_lcy, sales_lcy, amount_lcy`). For each such key the database keeps a totals table `company$Table$SIFT$key` (key columns as primary key, one column per sum field, `cnt`), maintained by **database triggers** generated by `backend/foundation/sift` (Postgres PL/pgSQL + SQLite triggers) in the same statement/transaction as the entry. Rules:
  - Triggers only apply differences (`sum = sum + x`, `ON CONFLICT DO UPDATE`): never read a total and write it back. Totals rows with `cnt = 0` are deleted. NULL key values are stored as blank (`''`, 0, false, blank date).
  - FlowFields (card `CalcFields`, list `CalcFieldsForRecords`, Sum and Count) automatically read the totals when a SIFT key of the source table covers all flow filter fields and the summed field (tablegen `resolveFlowFieldSIFT`); otherwise they add up the entries and tablegen prints a warning if the source table has SIFT keys.
  - Startup sync (`SyncKeys`, called by `CreateTableWithDBType` and table sync) builds/rebuilds totals when the key definition's fingerprint changed (fields added/removed), when the totals table is missing, and drops totals of removed keys; definitions are stored in the global `_sift_definition`. Each rebuild is one transaction; on Postgres an advisory lock lets only one instance build. Deleting a company calls `sift.DropCompany`.
  - Object names over 63 bytes are shortened with a hash (`sift.ObjectName`) — never build SIFT object names by hand.
  - Tests: `go test ./backend/foundation/sift/` runs on SQLite; set `TEST_POSTGRES_DSN=postgres://openerp:openerp@localhost:5432/sift_test?sslmode=disable` (database `sift_test` in the local docker Postgres) to also run them on Postgres, incl. the concurrency test.
- **Verify SIFT (codeunit 902)**: run from the Job Queue in a company; compares every SIFT totals table with a fresh `GROUP BY` over its entries (`sift.VerifyKey`, generated `VerifySIFT`). Parameter `VERIFY` (default) reports — differences fail the run, so a scheduled job goes to Error and can e-mail; `REPAIR` rebuilds the keys that differ (`sift.RebuildKey`, one transaction, entry table locked against writes on Postgres). SIFT totals rows are updated on every posting, so on Postgres totals tables are created `WITH (fillfactor = 50, …)` plus per-table autovacuum thresholds (`sift.storageOptions`; `Sync` adds them to older tables with `ALTER TABLE … SET`): the updates are HOT and old row versions are pruned without waiting for autovacuum. `RebuildKey` runs `VACUUM ANALYZE` afterwards when it owns the transaction.
- **Bulk loads**: insert many records with the generated `InsertAll(records, runTrigger)` (multi-row INSERT per batch via `ftables.InsertRows`; each record's OnInsert runs first), never `Insert` in a loop — one round trip per record made HEAVY demo data take minutes. For entries with SIFT keys call `sift.Suspend(tx, …, table)` before the load and the table's `SyncKeys(tx, …)` after it, in the same transaction: the totals are then built once instead of updated per entry (no dead row versions).
- **FlowFilter fields (FieldClass FlowFilter)**: a table field with `flow_filter: true` (types `types.Date`, `bool`, `int`, `types.Text`, `types.Code`) is not stored; it holds a filter expression the user sets, e.g. Customer `date_filter`. A FlowField applies it with a flow filter of type `filter`: `{field: posting_date, type: filter, value: date_filter}` — only when the FlowFilter has a value. Rules:
  - The generated record keeps each FlowFilter's expression; set it with `SetFlowFilter(field, expr)` (validated by `backend/foundation/flowfilter`, which builds the SQL with bind parameters: `a..b`, `..b`, `a..`, `|`, `<>`, `*` for Text/Code, Yes/No). The API takes `flow_filters` (JSON `[{field, expression}]`) on `/list` and `/card`; page definitions send `flow_filter_fields` (name, kind, caption).
  - Dates reach the backend as ISO only. The user types filter syntax in the filter pane's "Filter totals by" section (`01.01.26..31.03.26`, `t` = today); `utils/flowFilter.ts` converts it (`toApiFlowFilter`, `apiFlowFilters`): dotted dates are always day.month[.year] (`01.04` = this year) whatever the UI language — dots only occur in day-first formats, and an en-US user typing `01.04.26` means 1 April; slash/dash dates follow the UI language's order; ISO always works. The empty field shows the user's format as a hint. The modal card opened from a list gets the list's FlowFilters.
  - SIFT: tablegen resolves two keys per FlowField — without and with the FlowFilter fields — and the generated code picks at run time; with no key covering the FlowFilter fields a filtered FlowField sums the entries (tablegen warns). Customer has no date SIFT key on purpose: with ~1–2 entries per customer and day a (customer, posting_date, open) key holds about as many rows as entries and gave no speed-up (measured on HEAVY demo data), while costing a totals update per posting. Add one only for tables with many entries per day.
- **Date formula fields**: a field that holds a date formula (`30D`, `1M+CM`, `-CY`, `WD5`) is a `types.Code` field (length 20) with `date_formula: true` in the table YAML — no separate field type. The generated field checks and `ValidateField` reject a value that is not a date formula (`ERR_DATE_FORMULA_INVALID`); business logic calculates with it through `system.CalcDate(string(rec.Due_date_calculation), date)`. The parser lives in `backend/foundation/dateformula`, shared by both — never write a second one.
- **Sensitive fields (ABSOLUTE RULE)**: a table field with `sensitive: true` in its YAML (e.g. `User.password_hash`) never leaves the server through the table API. Handlers send records with `ftables.PublicMap(table)`, never `table.ToMap()` (that is for internal use: keys, comparisons); field names from the client in filters, `sort_by` and `search_fields` are checked with `ftables.IsQueryableColumn` (a filter or search on a hash reveals it piece by piece); insert/modify/validate reject a payload that sets a sensitive field (`rejectSensitiveFields`). Tablegen refuses a sensitive primary key/FlowField and a table relation whose dropdown shows a sensitive field. User passwords are set through the virtual `password` field (hashed by `SetPassword`). Mark every new secret or hash field `sensitive: true`, and never put one on a page.
- **Masked fields (ABSOLUTE RULE, write-only secrets)**: a secret the user must be able to set but never read back (e.g. `SMTP_Setup.password`) is `masked: true` in the table YAML (Text/Code only). `PublicMap` sends `ftables.MaskedValue` (`••••••••`) when it has a value, "" when empty; writes go through `DropMaskedPlaceholders`, so the placeholder sent back keeps the stored value and "" clears it; like sensitive fields it cannot be filtered, sorted or searched. Server code (e.g. the mailer) reads the real value from the table record. A masked secret the server must read back in clear (it sends it on, e.g. the SMTP password) is also `encrypted: true` in the table YAML (requires `masked`, `types.Text`): it is **stored encrypted** (`backend/foundation/secrets`, AES-256-GCM, `enc:v1:…`; key `OPENERP_ENCRYPTION_KEY`, else derived from `JWT_SECRET`). The generated Insert/Modify/InsertAll encrypt it (and refuse a value too long to fit the column encrypted — 154 bytes for length 250), the generated `Plain<Field>()` decrypts it for the server code that uses it, and startup encrypts values stored before (`tables.EncryptStoredSecrets`). Every new stored secret of this kind gets `encrypted: true` — never decrypt one into an API response. Passwords that are only checked (user logins) are hashed instead (`sensitive: true`, bcrypt), never encrypted. The page metadata marks masked fields (`field.masked`, from the table YAML or a page field's `masked: true` — e.g. the User card's virtual `password`, hashed by the API) and the frontend renders them as password inputs on cards, list cells and modal cards — decide by `field.masked`, never by the field name. Use `sensitive` for values nobody may set or see (hashes), `masked` for secrets an admin enters.
- **Keys inside panels**: list shortcuts (`use:shortcuts` on the list page) skip keys from inside an element marked `data-own-keys` (the filter pane). Svelte 5 delegates `onkeydown` to the app root, so a `stopPropagation` in a component's handler runs after the list's native shortcut listener — it cannot stop it.
- **Company display name**: `Company.name` is the technical key — it prefixes every company table, so it allows only `a-z 0-9 _` (lowercased on create). `Company.display_name` (Display Name, e.g. "Demo Company 01") is what users see. `GET /api/auth/companies` returns `{name, display_name}` objects; the session, JWT and `POST /api/auth/company` use `name`. In the frontend show companies via `companyLabel`/`companyLabelFor` (`utils/company.ts`), never the raw name.
- **Renaming / deleting a company (Rename)**: changing `Company.name` (Companies page) moves the company's data. The generated `Modify` runs the `OnRename` trigger whenever a primary key field changes (every table; `OldValue(field)` gives the xRec value). `Company.OnRename` validates the new name (`company.NormalizeName`: lower case `a-z 0-9 _ -`, 2–50), refuses an existing company, then `company.RenameObjects` drops the old SIFT objects (their trigger functions name the totals tables) and renames every `old$…` table, index and sequence, and `syncCompanyKeys` rebuilds indexes and SIFT totals for the new name; the generated cascade updates User Member. All of it runs in the API's modify transaction (`RenameObjects` refuses a plain `*sql.DB`) — a failure at any step rolls everything back. Deleting a company drops its tables and SIFT objects in the delete's transaction (`company.DropObjects`). Find a company's objects only by exact name prefix (`company.Tables`), never `LIKE 'name$%'` — `_` is a LIKE wildcard. Afterwards `AuthHandler.CompanyChanged` drops the company's cached sessions and re-issues the renaming user's cookie; any other token naming a company that no longer exists is treated as logged out by the auth middleware. At startup a missing `COMPANY_NAME` company is only created on an empty database; otherwise an existing company is entered.
- **Trigger and validation messages (ABSOLUTE RULE)**: never `errors.New("…")` with English text in a table trigger. Field checks use `ftables.CheckRequired` / `CheckMaxLength` (counts characters, not bytes) / `CheckRange`; other rule violations return an `apperrors` error that names the field (`FieldTooShort`, `FieldFormat`, `RelatedNotFound`, `RelatedInactive`, …): `AppError{Table, Field}` puts the field's caption in the user's language into the message (`errors.yaml`, every language). Handlers return trigger errors with `errorMessage(err, language)`, never `err.Error()`. The generated `CheckFields` follows this.
- **Index names**: a company table's index is `company$Table$key`; a global table's index is `Table$key`, shared by all companies (the old company prefix created one copy per company — removed by migration 007).
- **Job Queue notifications**: after a scheduled run the scheduler e-mails the outcome per `notify_on` to `Job_Queue.notification_email` — one or more addresses separated by `;` (checked and tidied by `OnValidate_Notification_email`, `mail.ParseAddressList`) — in the job's `notification_language` (blank = default language). Message building, recipients, language and the SMTP config live in `backend/business-logic/jobqueue`, shared with codeunit 455 "Send Test E-mail" (Job Queue list/card action). `mail.Sender.Send(to []string, …)` sends one message to all recipients; a rejected recipient comes back in a `*mail.RecipientsError` and does not stop the others. Notification failures are logged, never fail the job.
- **Demo data**: codeunit 901 (`codeunits/demodata_create.go` → package `business-logic/demodata`) fills the current company from `demodata/data/*.yaml`, run from the Job Queue (Run or scheduler); the job's `parameter` is `SMALL` (default) or `LARGE`. One transaction, fixed random seed, refuses a company that already has customers. Records go through `Insert(true)` — ledger entries through `InsertAll(…, true)` with their SIFT totals suspended and built once at the end — so table triggers apply (Customer `OnInsert` sets Status Open — Blocked is applied with a Modify afterwards). When a new business table gets demo data, add it to the YAML and `demodata.Create` in the same change.
- **FlowFields**: Tables can define computed aggregate fields from related tables. Defined in table YAML with `flow_field: true`, `calc_formula` (Sum, Count), `source_table`, `source_field`, and `flow_filters`. The tablegen tool generates `CalcFields()` methods. The API automatically calls `CalcFields()` for list and get endpoints when flow fields are in the requested fields. Flow fields are read-only and not persisted to the database. Example: `Customer.balance_lcy` sums `CustomerLedgerEntry.remaining_amt_lcy`, `Job_Queue.number_of_entries` counts related `Job_Queue_Entry` records. Lists must not call `CalcFields` per row (one query per row and FlowField — 28 s for 10,000 customers): `ListRecords` reads the rows, then calls the generated `CalcFieldsForRecords(records, fields...)`, which runs one grouped query per FlowField (IN list for ≤500 keys, whole source table above) and only for the requested fields. Keep that pattern for any endpoint returning many records. Lists can **filter** on Sum/Count FlowFields (`balance_lcy` `>10000`): the generated `SetFilter` puts the filter (`flowfilter`, kinds decimal/int, also `<`/`>`/`<=`/`>=`) on a correlated subquery (`flowFieldExpr`) that reads the same source as `CalcFieldsForRecords` — SIFT totals when a key covers the filters; the API validates the expression (`FlowFieldFilterKind`, `ERR_FILTER_INVALID`). Not sortable.

## Page Definition Guide

All pages are defined in backend YAML files at `backend/definitions/pages/`. File naming convention: `{ID}-{name}.yaml`. Pages auto-load from this directory — no Go registration needed.

### List Page YAML Structure
```yaml
page:
  id: 22                      # Unique page ID (base application: 1-999, extensions: their series in application.yaml)
  type: List                   # Must be "List"
  name: Customer List          # Internal name
  source_table: Customer       # Table to read from (its registry name: YAML name or api_name)
  caption: Customer List       # Display title (translated via pages.yaml translation files)
  card_page_id: 21             # (optional) Associated card page for row click / Edit action
  modal_card: true             # (optional) Open card in modal instead of navigating to card page
  editable: true               # (optional) Enable inline editing in the repeater
  delayed_insert: true         # (optional) DelayedInsert: insert a new row only when leaving the row

  layout:
    repeater:
      fields:
        - source: no           # Field name from source_table
          width: 120           # Column width in pixels
          caption: Phone No.   # (optional) Override caption (otherwise uses translation)
        - source: number_of_entries
          width: 120
          drilldown: 673                        # (optional) Page ID to navigate to on click
          drilldown_filter_field: job_queue_no   # Field on target table to filter by
          drilldown_filter_value: no             # Field on current record for filter value

  actions:                     # Toolbar buttons
    - name: Run                # Action name (used by handleAction)
      caption: Run             # Button text (translated via translation files)
      shortcut: F9             # (optional) Keyboard shortcut
      promoted: true           # (optional) Show in toolbar (non-promoted go in overflow menu)
      enabled: true            # (optional) Default enabled state
      run_page: 25             # (optional) Navigate to page ID on click
      run_object: codeunit:900          # (optional) Run codeunit by ID
      run_object: field:object_id_to_run # (optional) Run codeunit from a record field value
      # Actions are enabled unless enabled: false (a missing value means enabled)
```

### Card Page YAML Structure
```yaml
page:
  id: 21
  type: Card                   # Must be "Card"
  name: Customer Card
  source_table: Customer
  caption: Customer Card
  list_page_id: 22             # (optional) Back-link to list page
  enable_navigation: true      # (optional) Show First/Prev/Next/Last navigation buttons
  editable: true               # (optional) Enable editing
  focus_field: "no"            # (optional) Auto-focus this field on page load

  layout:
    sections:                  # Card uses sections instead of repeater
      - name: General
        caption: General       # Section header (translated via translation files)
        fields:
          - source: no
            editable: true     # Per-field editability
            importance: Promoted  # (optional) Visual prominence
          - source: balance_lcy
            editable: false    # Read-only field
            style: Strong      # (optional) Visual styling (bold)
          - source: payment_terms_code
            editable: true
            table_relation: Payment_terms  # (optional) Enable lookup dropdown

  actions:                     # Same structure as list page actions
    - name: Back to List
      caption: Back to List
      shortcut: Esc
      promoted: true
      run_page: 22
    - name: Ledger Entries
      caption: Ledger Entries
      shortcut: Ctrl+F7
      promoted: true
      run_page: 25
      run_page_filter_field: customer_no   # (optional) open run_page filtered:
      run_page_filter_value: no            # customer_no = this record's no
    - name: Generate Ledger Entries
      caption: Generate Ledger Entries
      promoted: true
      run_object: codeunit:900
```

### Key Rules
- **Card actions**: only `promoted: true` actions are shown as buttons on a card (there is no overflow menu); others are reachable by their shortcut only. `run_page_filter_field` / `run_page_filter_value` open `run_page` filtered to the current record (like list drilldowns).
- **Table names**: pages use registry names (`Customer_ledger_entry`), table YAML files display names (`Customer Ledger Entry`). Table metadata (`TableMetadata`, primary key fields, field types) normalizes both (`metadataKey`); a page whose source table finds no metadata gets no `primary_key_fields`, and its rows are keyed randomly (focus lost on every render).
- **Lookup field definitions**: `table_relation` with `lookup_columns` is defined in the **table** YAML (`backend/definitions/tables/`), not the page YAML. The page YAML only needs `table_relation: TableName` on the card field. The backend resolves lookup column data automatically and sends it in the `captions.lookups` response.
- **Field captions**: Come from `translations/{lang}/tables.yaml` by default. The page YAML `caption` field overrides the translation for a specific page only.
- **Page captions**: Come from `translations/{lang}/pages.yaml`. The YAML `caption` is the fallback if no translation exists.
- **Action and section captions**: come from `translations/{lang}/common.yaml`, `common.actions.<name>` / `common.sections.<name>` with the YAML `name` in snake case (`Send Test E-mail` → `send_test_e_mail`); the YAML `caption` is the fallback. `TestPageCaptionsTranslated` fails when a page uses a name without a translation in every language — add it to all `common.yaml` files. The frontend decides by `action.name`, never by the caption.
- **Option captions**: the table API sends option values in the user's language (`optionCaptions`: `tables.<table>.options.<field>.<value in snake case>`, "On Hold" → `on_hold`) under the option index; records carry the index, so the frontend shows the caption but always saves the index. `TestOptionCaptionsTranslated` fails when an option has no caption in every language — add it to all `tables.yaml` files (a blank option " " needs none).
- **Page definitions are shared**: `registry.GetPage` returns the registry's definition; the page handler works on `pageDef.Clone()` before translating or flagging anything — never change the shared one per request.
- **Page navigation (ABSOLUTE RULE)**: pages open **inside the app** with `goto()` (main menu, drilldowns, card actions, Esc/close, card record navigation, list ↔ card) — never `window.location.href`: a full page load restarts the whole app (JavaScript, session, translations) and is slower. Both `/pages` routes wrap `PageRenderer` in `{#key data}`, so every navigation, also between two `/pages/…` URLs, gets a new `PageRenderer` (it loads its page in `onMount`). URL updates that are not navigations (a card after insert/rename) use `replaceState` from `$app/navigation`, never `history.replaceState`. Full loads only where everything must start fresh: login, company switch, language change.
- **Frontend routing**: `PageRenderer.svelte` checks `page.type` and renders `<ListPage>` or `<CardPage>` — never create page-type-specific routes or components.
- **Modal card**: When `modal_card: true` on a list page, the Edit/New actions open a `<ModalCardPage>` overlay instead of navigating to a separate card page. The modal auto-saves on field changes and refreshes the list on close.
- **Drilldown fields**: List page fields can have `drilldown` (target page ID), `drilldown_filter_field` (field on target table), and `drilldown_filter_value` (field on current record). Clicking the cell navigates (`goto`) to `/pages/{drilldown}?filter={filter_field}={value}&back=…`. **Return**: drilldowns and card `run_page` actions add `back=<origin page with select=<record key>>` (`withReturn` in `utils/returnUrl.ts`, validated by `safeReturnUrl`). A list opened with `back` shows a close (×) button and Esc in navigation mode returns there instead of to the main menu; the origin list then opens on the window around the selected record and selects it (`initialSelect`, located via `/ids` when the list is unfiltered). Navigation between pages is client-side (see "Page navigation" below).
- **Card navigation follows the list**: First/Previous/Next/Last on a card move through the records the list showed, in its order. `GET /api/tables/:table/ids` takes the list's `filters`, `search`/`search_fields` and `sort_by`/`sort_order` (`applyListQuery`, shared with `/list`) and returns full keys (comma-joined for composite keys). The list passes its query (`navigationQuery`, `utils/recordNavigation.ts`) to the modal card as a prop and to a card page as `?nav=<JSON>`, which card navigation and pop-out keep.
- **URL filter parameter**: List pages accept `?filter=field=expression` query parameter to pre-filter on load. Parsed in `PageRenderer.svelte` into `currentFilters` and passed to the API. Used by drilldown links to show related records.

## Frontend Conventions

- **TypeScript strict mode** with `checkJs` enabled
- **Suppressed Svelte warnings**: `a11y_no_noninteractive_tabindex`, `a11y_no_noninteractive_element_interactions` (intentional tabindex patterns), `css_unused_selector` (Tailwind `@apply`)
- **Colors are theme variables (ABSOLUTE RULE)**: every color is a CSS variable holding an RGB triplet (`--gray-900: 18 18 18`), defined in `src/app.css` and used as `rgb(var(--x))` / `rgb(var(--x) / 0.5)` — a theme overrides them on `<html>` at run time, so a hardcoded color does not follow the theme. Tailwind's colors point at them (`tailwind.config.js`): `gray-*` (`--gray-*`), `blue-*`/`primary-*` (`--accent-*`), `surface` (panels, cards, dropdowns; use `bg-surface`, never `bg-white`), `page` (page background), `fg` (body text), `heading` (titles, light mode), `appbar`/`appbar-fg` (app bar). Semantic variables: `--row-selected`/`--row-selected-hover` (selected list rows), `--text-sel-bg`/`--text-sel-fg` (selected-text look of list cells), `--scroll-*` (scrollbars), `--font-sans`. The defaults are the classic palette (reference screenshots kept locally, not in the repository): the gray ramp serves both modes (light page white, text `#212121`, secondary `gray-600`; dark page `gray-900` `#121212`, text `gray-50`, secondary `gray-400`, separators `gray-700`), the accent is teal (links `600` light / `400` dark, tiles `500`), the app bar `#282828` in both modes. In `<style>` blocks write `rgb(var(--gray-700))`, never the hex value; never introduce other hues (no indigo/purple/Tailwind blue) except red/green/yellow for status. Dark mode via `class` strategy (`.dark` on `<html>`), which switches the mode-specific variables (`--page-bg`, `--fg`, `--row-selected`, …) in `app.css`. Font: `--font-sans` (Segoe UI).
- **Themes and appearance (per user)**: the user menu's "Appearance" dialog (`menu/AppearanceDialog.svelte`) lets each user choose a theme for light mode and one for dark mode; the sun/moon button switches between the two. Stored per user in the global table `User_Personalization` (2000000073; also wallpaper, font, density), whose triggers check every value (theme exists and fits its slot's mode — `ERR_THEME_WRONG_MODE` — and the ranges); `GET/PUT /api/appearance` (`handlers/themes.go`; PUT changes only the fields sent) and `GET /api/themes`. A user without a record gets Light/Dark and keeps the mode the browser had (`saved: false`). Themes are **colors only** — never CSS, markup or URLs: `backend/foundation/themes` parses palettes (Omarchy `colors.toml` in both forms, `alacritty.toml`) strictly (`ParseColor`), normalizes them into tokens (`Normalize`: gray ramp from background to foreground in OKLab with lightness profile, accent ramp with teal's, contrast guard — text ≥ 7:1, secondary text/links/selection ≥ 4.5:1, the background moves when it cannot carry readable text) and stores `#rrggbb` only. Built-in themes: Light/Dark (tokens equal to the `app.css` defaults, checked by `TestClassicTokensMatchAppCSS`) plus `themes/builtin/*.toml`. The frontend (`lib/theme/apply.ts`, `stores/theme.ts`) turns tokens into the CSS variables on `<html>` (only `#rrggbb` → `r g b`, only known names) and caches both slots in localStorage `theme-cache`, which the inline script in `app.html` applies before the first paint (it accepts only `--name: r g b` pairs).
- **Themes from GitHub (ABSOLUTE RULE: untrusted input)**: every logged-in user installs Omarchy theme repositories by URL in the Appearance dialog for their own themes (`POST /api/themes/install`, update `POST /api/themes/:code/refresh`, remove `DELETE /api/themes/:code`; endpoints, not codeunits). Deleting a theme for everybody is the Themes page (generic table API, Delete permission on `Theme`). `business-logic/themerepo` installs, `foundation/themes/github.go` downloads, stored once in the global tables `Theme` (901, tokens JSON + repo, pinned `commit_sha`) and `Theme_Asset` (902, images, `data` is `sensitive`). **Installed themes are per user**: `User_Theme` (903) holds the themes each user added — the dialog lists, the user can choose (`UserThemeByCode`, checked by `User_Personalization`) and the asset endpoint serves only those; installing a repository someone else installed only adds it to the user's themes (no download); Remove (`DELETE /api/themes/:code`) takes it off the user's themes and deletes the files when no user has it any more; the Themes page (Delete permission) deletes a theme for everybody. At startup `themerepo.LinkInstallers` gives themes without users to the user who installed them. Rules — keep them when changing this code: only `https://github.com/<owner>/<repo>[/tree/<ref>]` (`ParseRepoURL`); the backend builds every URL itself, for api.github.com and raw.githubusercontent.com only, redirects to other hosts refused, `GITHUB_TOKEN` sent to the API host only; only `colors.toml`, `alacritty.toml`, `preview.png`/`theme.png` and `backgrounds/*.{jpg,png,webp}` are downloaded (never scripts, CSS, SVG, archives); every body through `LimitReader` (theme file 64 KB, image 6 MB, 30 MB total, ≤ 5 wallpapers, tree ≤ 2000 entries, 90 s); images are checked (magic bytes, `DecodeConfig` ≤ 8000 px / 20 MP before decoding) and **re-encoded** (JPEG/PNG, scaled) — the downloaded bytes are never stored or served; `GET /api/themes/:code/assets/:name` sends them with `nosniff`, `Content-Security-Policy: default-src 'none'; sandbox`; colors go through `ParseColor`/`Normalize`, stored tokens through `ParseTokens`; one install at a time, 5 per minute, at most 50 themes; `THEME_ALLOWED_OWNERS` limits the owners. Theme names/authors are shown as text only, never `{@html}`. Tests: `foundation/themes/*_test.go` (fake GitHub: redirects, gzip bomb, oversized and polyglot files, pixel bombs) and `themerepo/install_test.go`. Admin overview: page 906 Themes (Settings).
- **Wallpaper (Appearance)**: `User_Personalization.wallpaper` is blank or `bg-1` … `bg-5` of the theme in use (a theme with fewer wallpapers shows its first; built-in themes have none), with `wallpaper_opacity` (0–100), `wallpaper_blur` (0–20 px) and `panel_opacity` (75–100, checked by the table). `stores/theme.ts` (`wallpaperFor`) sets `has-wallpaper` and `--wp-opacity`/`--wp-blur`/`--panel-alpha` on `<html>`; `+layout.svelte` renders the fixed `.wallpaper-layer` behind the page, blended over the solid page background. Rules in `app.css`: the page containers become transparent and the panels (cards, list rows, status bar, page header, menu bar, breadcrumb, filter pane) take `--panel-alpha`; dropdowns, dialogs, the sticky column header and the selected row stay opaque. Never put `backdrop-filter` (or anything else that creates a stacking context) on a panel: it traps the dropdowns inside it (the user menu went behind the card header). New page surfaces that should show the wallpaper go into that selector list.
- **Font, font size, density (Appearance)**: `User_Personalization.font` (Default = Segoe UI stack, System, Inter, IBM Plex Sans, Serif, Monospace), `font_size` (0 = default, else 80–150 %; the dialog offers 90/110/125) and `density` (Comfortable = the list rows, Compact = no cell padding). `layoutOf`/`applyLayout` (`lib/theme/apply.ts`) put `font-<name>` / `density-compact` classes and `--font-scale` on `<html>` (also from the `theme-cache` before the first paint); `app.css` maps them to `--font-sans`, `html { font-size }` and the list cell variables `--cell-py`/`--cell-lh`. Fonts are self-hosted (`@fontsource-variable/inter`, `@fontsource/ibm-plex-sans`, imported in `+layout.svelte`; a file loads only when the font is used) — never load fonts from Google or another service. Sizes in components should be in rem so they follow the font size; the list measures its rows (`offsetHeight`), never assume a fixed row height.
- **Loading skeleton**: `PageRenderer` shows the list/card skeleton only when loading takes longer than 1 second (`skeletonDelay`); a quick load shows no placeholder flash.
- **Svelte 5 runes**: Always use runes (`$state`, `$derived`, `$effect`, `$props`, `$bindable`) — never legacy Svelte 4 patterns (`export let`, `$:`, `$store` syntax).
- **i18n**: All labels and field captions in the frontend must come from backend translation files — never hardcode display text in Svelte components. In backend codeunits, use `i18n.GetInstance().Message(key, CurrentLanguage())` for user-facing strings (dialog titles, field labels, messages). Never hardcode English strings that are shown to the user. Never use `t(...)` as a `$props()` default or a `$state()` initial value: those are evaluated once at mount, possibly before translations have loaded, and then show the raw key (e.g. `BTN_CANCEL`) — look the default up in the template instead (`{cancelText ?? t(BTN.CANCEL)}`).
- **keyboard shortcuts**: Alt+N new (as in the web client — never Ctrl+N, Ctrl+T or Ctrl+W: browsers reserve them and never pass them to the page), Ctrl+E edit, Ctrl+D delete, Ctrl+S save, Ctrl+F find, Escape cancel (there is no Refresh action: F5 is the browser's reload; lists reload themselves after their actions), Ctrl+Home/End first/last, PageUp/Down prev/next, Ctrl+O switch company (a global capture-phase handler in `MenuBar` opens `CompanySwitchDialog` on every page, even inside a list cell, and blocks the browser's Open File dialog — browsers do let the page take Ctrl+O). Switching company (dialog or user menu) always lands on the main menu (`/`, full load), never reloads the current page. While the dialog is open, `companySwitchOpen` is true and window-level page key handlers (ListPage) must ignore keys
- **Keyboard Shortcuts help**: the user menu's "Keyboard Shortcuts" item opens `/help/shortcuts` as a detached window (`window.open` with the fixed name `openerp-shortcuts`, so it is reused). The root layout renders `/help/*` routes without MenuBar/Breadcrumb; Esc closes the window. Its content is `shortcutHelpSections` in `utils/shortcutHelp.ts` (translation keys `HELP_*` in `messages.yaml`, checked for every language by `shortcutHelp.test.ts`). When adding or changing a shortcut, update that list too.
- **Code field behavior (ABSOLUTE RULE)**: Fields with `types.Code` must allow typing in any case, then auto-uppercase the value on blur (when the field loses focus). This matches standard behavior. Never force uppercase while typing — only convert on exit. Apply this everywhere Code fields are rendered: login forms, list page cell-editing, card page fields, modal dialogs, and any other input bound to a Code field.
- **Boolean field rendering (ABSOLUTE RULE)**: Fields with YAML `type: bool` must always render as checkboxes — on card pages, list pages (edit and read-only mode), and modal card pages. Detection uses two complementary checks: `typeof value === 'boolean'` (works for existing records) OR `fieldTypes[field.source] === 'bool'` (works for new/empty records where value is `undefined`). The `fieldTypes` metadata flows from backend YAML → `TableMetadata` → page API response `captions.field_types` → `PageRenderer` → `CardPage`/`ListPage`/`ModalCardPage` → `FieldRenderer`. Never rely solely on `typeof` — new records have no value yet, so the backend metadata is essential. This is fully generic: any table field with `type: bool` in its YAML definition automatically gets checkbox rendering everywhere. On editable list pages, boolean checkboxes are **clickable in edit mode only** (cell-selected / cell-editing, after Edit / Ctrl+E / F2) — clicking toggles the value and saves immediately (`handleCellBlur`, or `api.modifyRecord` for a checkbox in another row shown read-only during edit mode); on error, revert the checkbox value. In navigation mode (the list is not being edited) a checkbox only shows its value — `disabled` with `pointer-events: none`, so the click reaches the cell and selects the row. On non-editable list pages, boolean checkboxes are always `disabled`.

- **Date formatting (ABSOLUTE RULE)**: All Date and DateTime values must be displayed using locale-aware formatting via `Intl.DateTimeFormat` with the user's session language. Date fields use `formatDate()` and DateTime fields use `formatDateTime()` from `fieldHelpers.ts`. Detection uses `isDateType(fieldTypes[field])` and `isDateTimeType(fieldTypes[field])` which handle both YAML types (`"types.Date"`) and Go FieldType constants (`"Date"`). The API wire format is always ISO (`YYYY-MM-DD` / RFC3339) — locale formatting is display-only. For card page editing, use `<input type="date">` / `<input type="datetime-local">` (browser handles locale). A `datetime-local` input shows nothing for an RFC3339 value with a zone and returns local time without one: convert with `toDateTimeInput` / `fromDateTimeInput` (`fieldHelpers.ts`) — done in `FieldRenderer` and list cell editing. For ProgressModal date dialogs, use text input with locale placeholder from `getDateFormatPattern()` and convert back to ISO via `parseLocaleDate()` on submit.

- **Copy/paste in list pages (ABSOLUTE RULE)**: Ctrl+C in cell-selected mode copies the cell's display value to the system clipboard via `navigator.clipboard.writeText()`. Ctrl+V in cell-selected mode pastes from clipboard, sets the cell value, and enters cell-editing mode. In cell-editing mode, the browser handles Ctrl+C/V natively on the input element — never intercept these keys in cell-editing mode. Boolean and non-editable fields ignore Ctrl+V.

- **Option field rendering (ABSOLUTE RULE)**: Fields with YAML `type: Option` must always render using the `OptionDropdown` component — on card pages (via `FieldRenderer`), list pages (cell-editing mode), and modal card pages. Never use native `<select>` elements for Option fields. The `OptionDropdown` component provides a custom dropdown with keyboard navigation: Alt+ArrowDown/ArrowDown opens the dropdown, ArrowUp/ArrowDown navigates options, Enter selects when dropdown is open, Escape closes, F4 toggles, Space opens when closed. **Critical**: Enter must only `preventDefault` when the dropdown is open (to select). When the dropdown is closed, Enter must NOT be handled — it must bubble up to `handleLookupCellKeyDown` for cell navigation (move to next row). Never re-open the dropdown on Enter when closed. In list page cell-selected mode, Option fields show the formatted value with a `▼` dropdown arrow button (same visual pattern as lookup fields). In list page navigation/read-only mode, Option fields display as plain text (the dropdown is available when entering cell-editing mode). The `options` metadata flows from backend YAML → `TableMetadata` → page API response `captions.options` → `PageRenderer` → components. In list page cell-editing mode, `OptionDropdown` is wrapped in a `<div data-row data-col>` container using `handleLookupCellKeyDown` for Tab/Enter/Escape/F2 cell navigation (same pattern as `LookupDropdown`). `focusCell()` finds the OptionDropdown via `[role="combobox"]` selector inside the container div.

- **Enter moves to the next field on card pages (ABSOLUTE RULE)**: Enter in a card field acts like Tab — focus moves to the next field in Tab order (text selected), Shift+Enter to the previous one, and leaving the field saves it exactly as Tab does. On the last field Enter stays put. Implemented once in `handleFieldEnterKey` (`utils/fieldNavigation.ts`) on CardPage's `.sections-container`, so it covers every card page and modal card. Controls that use Enter themselves (an open Option/Lookup dropdown selecting a row, a lookup rejecting typed text) `preventDefault` it and the handler leaves it alone — keep that contract when adding field controls. Textareas keep Enter for new lines.

## Generic List Page Behaviors (ABSOLUTE RULES)

All list page behaviors are driven by page metadata — never add table-specific logic in `ListPage.svelte` or `PageRenderer.svelte`.

### Three Cell States (Spreadsheet Model)

The list page uses a spreadsheet-style 3-state cell model (like Excel/LibreOffice Calc), not a simple 2-mode toggle.

1. **Navigation mode** (default) — No cell is focused. Whole rows are selected/highlighted. Arrow keys move row selection. This is the default state when the page loads.
2. **Cell-selected** — A single cell has a visible selection indicator (e.g., blue border) but the user is NOT typing in it. The cell content is displayed but not editable. Arrow keys move the selection to adjacent cells. The cursor is not visible.
3. **Cell-editing** — The cursor is inside the cell and the user is actively typing. Arrow keys move the cursor within the text (unless at a boundary). The cell contains a live input element.

#### State Transitions
- **Navigation → Cell-selected**: press F2 or Ctrl+E / the Edit action (lists without a card page; with one, Edit opens the card), or Enter on a list without a card page. Editing starts in the column clicked last (`editColumn()`), else the first. **A single click only selects the row** — it never starts editing.
- **Cell-selected → Cell-editing**: Press F2 (cursor at end, content preserved), or start typing a printable character (clears cell, enters typed character), or press Backspace (clears cell, enters editing), or double-click the cell.
- **Cell-editing → Cell-selected**: Press F2 (keeps current value) or Escape (reverts to value before editing began).
- **Cell-selected → Navigation**: Press Escape (reverts unsaved changes in the cell).
- **Cell-editing → Navigation**: Not direct — must go through cell-selected first (Escape twice: first reverts edit, second exits to navigation).
- **Moving between cells** (via Arrow/Tab/Enter in cell-selected or cell-editing): The leaving cell's value is confirmed (saved), and the destination cell enters cell-selected state.

### Keyboard: Navigation Mode

| Key | Action |
|-----|--------|
| ArrowUp / ArrowDown | Move row selection up/down |
| Home / End, Ctrl+Home / Ctrl+End | Select the first / last record of the whole list (loads that window) |
| PageUp / PageDown | Move row selection one page (the rows that fit in the visible list) up/down |
| Enter | If `card_page_id` set: open card page. Otherwise: enter cell-selected on first editable cell |
| F2 | Enter cell-selected on first editable cell of selected row |
| Ctrl+E | Enter cell-selected (same as F2) |
| Alt+N / Ctrl+Insert | Insert new row |
| Ctrl+F | Focus search input and select all text |
| Ctrl+F7 | Show/hide the filter pane (Table Filter); also works from inside the pane. The pane opens with the field picker focused (dropdown closed): Alt+↓, arrows, Enter adds the field (arrows on the closed picker only move the highlight) and the cursor goes to its value box; Enter applies and the cursor stays there. Esc in the pane closes the pane, not the list |
| Ctrl+D | Delete the selected record — with several rows selected, all of them (one confirmation, one transaction) |
| Shift+ArrowUp / ArrowDown / PageUp / PageDown / Home / End | Select the rows from the anchor to the new cursor row |
| Ctrl+A | Select every record of the list (all matching the filters and search, also those not loaded) |
| Ctrl+click / Shift+click | Add/remove a row / select the rows from the anchor to the clicked row (stays in navigation mode, also on editable lists) |
| Click and drag | Select the rows from the pressed row to the row under the mouse; near the top/bottom edge the list scrolls on (`handleRowMouseDown` / `dragAutoScroll`); a plain click selects one row |
| Ctrl+C / Ctrl+V | Copy the selected records / paste the copied records — on lists with `copy_paste_allowed: true` only (see below) |
| Escape | With several rows selected: back to the cursor row; otherwise navigate to home (`/`) |

### Multi-row selection (ABSOLUTE RULE)
- Every list supports it, in **navigation mode only**: `selection` (`utils/selection.ts`, unit-tested) holds record **keys** (`getRecordId`), never row positions — windows, search and sort move rows. `all` (Ctrl+A) means every record matching the list's query, also those not loaded; Shift ranges past the loaded window take the keys from `/ids` with the list's query (`selectionQuery`: filters, FlowFilters, search, sort).
- With one record or none selected the list behaves as before (actions use the cursor row, `selectedIndex`). Plain moves and plain clicks collapse a multi-selection; entering a cell (`enterCellSelected`) clears it; a new filter, search or sort clears it; Esc collapses it before it would close the list.
- Delete acts on the selection (`handleBulkDelete` → `PageRenderer.handleListBulkDelete` → `DELETE /api/tables/:table/delete-many` with `{keys}` or `{all: true}` plus the list's query from `listQuery()`, the same query as the list windows): one transaction, each record's OnDelete trigger, the first refusal rolls back all and names the record; at most 10,000 records. Other actions (Edit, Run, …) always use the cursor row — running them on a selection is not wanted (decided 2026-10-11).
- Selected rows use the `selected` row style; the cursor row of a multi-selection gets `cursor-row`. Ctrl/Shift+mousedown is `preventDefault`ed so the browser does not select text.

### Copy/paste of records (page `copy_paste_allowed: true`, CopyPasteAllowed)
- Navigation mode only (in cell modes Ctrl+C/V copy a cell value, and in inputs or with selected text the browser handles them). Handled in the window-level key handler, not the list's element shortcuts, so they work without focus on the list (pasting into an empty list). Ctrl+C copies the selected records (multi-selection, Ctrl+A = all of the list's query, else the cursor row): `POST /api/tables/:table/copy` returns their stored fields (`copyableMap`: no FlowFields, FlowFilter fields, sensitive or masked/encrypted fields), kept with the table and company in the browser tab (`utils/recordClipboard.ts`, sessionStorage — it survives a company switch).
- Ctrl+V pastes into the same table only: each primary key field the list is filtered on with one exact value replaces the copied value (`keyValuesFromFilters` — e.g. User Members filtered on role CLERK), otherwise keys stay (pasting into another company). One confirmation (a "Paste" button, `confirm.show(…, {confirmText, variant})`), then `POST /api/tables/:table/insert-many`: one transaction, every record through `insertOne` — the same steps as a single insert (relations, OnValidate, duplicate check, OnInsert); the first failure rolls back all and is named ("Record 2 of 3 cannot be pasted: …"). At most 1,000 records; Company is never pasted. The pasted records are selected afterwards. The permission check maps `/copy` to read and `/insert-many` to insert.
- Pasting from Excel or between different tables is out of scope (decided 2026-10-12).

### Keyboard: Cell-Selected Mode

| Key | Action |
|-----|--------|
| ArrowUp / ArrowDown | Confirm value + move selection to cell above/below. ArrowDown on last data row: create new row |
| ArrowLeft / ArrowRight | Move cell selection left/right within the row |
| PageUp / PageDown | Confirm value + move selection one page up/down in the same column |
| Tab | Confirm value + move selection right (wraps to next row; last column of the last row: create new row) |
| Shift+Tab | Confirm value + move selection left (wraps to previous row) |
| Enter | Confirm value + move selection down. On last data row: create new row |
| F2 | Enter cell-editing (preserve content, cursor at end) |
| Escape | Revert unsaved changes in cell, return to navigation mode |
| Delete | Clear cell content (set to empty string), stay in cell-selected |
| Backspace | Clear cell content + enter cell-editing |
| Printable character | Clear cell content + enter cell-editing with typed character |
| Ctrl+C | Copy cell value to system clipboard |
| Ctrl+V | Paste from clipboard into cell + enter cell-editing |
| F8 | Copy value from the cell directly above, then enter cell-editing with the cursor at the end (standard). Escape reverts to the value before F8 |
| Alt+N / Ctrl+Insert | Insert new row |
| Alt+ArrowDown (on lookup cell) | Enter cell-editing and open lookup dropdown |
| Space (on boolean cell) | Toggle checkbox value |
| Enter (on boolean cell) | Toggle checkbox value + move selection down |

### Keyboard: Cell-Editing Mode

| Key | Action |
|-----|--------|
| ArrowUp / ArrowDown | If cursor at text boundary (start/end) or all text selected: confirm + move selection. Otherwise: move cursor in text |
| ArrowLeft / ArrowRight | Move cursor within text. At text boundary: confirm + move selection to adjacent cell |
| PageUp / PageDown | Confirm + move selection one page up/down in the same column |
| Tab | Confirm + move selection right (wraps to next row; last column of the last row: create new row) |
| Shift+Tab | Confirm + move selection left (wraps to previous row) |
| Enter | Confirm + move selection down. On last data row: create new row |
| F2 | Exit cell-editing → return to cell-selected (keep current value) |
| F8 | Copy value from the cell directly above, cursor at the end of the text |
| Escape | Revert cell to value before editing began, return to cell-selected |
| Ctrl+C | Native browser behavior (copies selected text) |
| Ctrl+V | Native browser behavior (pastes at cursor) |
| All other keys | Normal text input behavior |

### Key Behavioral Notes
- **"Confirm"** means: save the current cell value. For existing records this triggers `modifyRecord`. For new records it validates the field and inserts the record once it is insertable (see Record Entry below) — on the row, not when the row is left.
- **Cell-selected visual**: The cell shows a distinct border/highlight (e.g., blue border) without a cursor. This must be visually distinct from cell-editing (which shows a cursor in an input).
- **Transition from navigation → cell-selected does NOT save anything** — it's purely a focus/selection change.
- **Lookup fields** (LookupDropdown, `<select>`) in cell-selected mode show the formatted value plus a **▼ dropdown arrow button**. Clicking the arrow or pressing Alt+ArrowDown enters cell-editing and opens the dropdown. The user can also enter cell-editing via F2, typing, or double-click. LookupDropdown manages its own keyboard internally (arrow keys navigate the dropdown list, Enter selects, F4 toggles, Escape closes). Cell navigation keys (Tab, Shift+Tab, Enter when dropdown closed, Escape when dropdown closed, F2) are handled by `handleLookupCellKeyDown` on the wrapper div, which intercepts events that bubble up from LookupDropdown.
- **`<select>` fields** (simple lookups) in cell-editing mode: arrow keys cycle through options natively (not intercepted by `handleCellKeyDown`). Tab/Enter handle cell navigation.
- **Boolean fields** (checkboxes) toggle on Space/Enter in cell-selected mode. They have no separate cell-editing state.

### New Record Creation
- New rows are created by `createNewRecord()`: all repeater fields start as `""` (so composite PK fields are always defined), then the row is filled with the table's defaults from `POST /api/tables/:table/init` (`InitRecord()`, `OnNewRecord`). If the init call fails the row stays blank — data entry is never blocked. Values the user typed before the defaults arrived are never overwritten.
- The values a row was initialized with are kept in `_pristine`. Only changes away from them count as user edits (`hasUserEdits`) — init-supplied defaults never do.
- Only one untouched new row can exist at a time — clicking New again focuses the existing one.
- New rows are marked with `_isNew: true` and a `_tempId` for stable keyed rendering.
- Editable lists render a **trailing blank row** after the last record **in edit mode** (only when the loaded window reaches the end of the list); clicking it starts a new record. In navigation mode it is not shown (a click must not start editing): New (Alt+N) starts a record. ArrowDown/Enter past the last record of the whole list does the same — "last" is decided by `isLastRow()` (window end and no rows beyond it per `total`), never by `displayRecords.length - 1` alone.
- If `card_page_id` is set with `modal_card: true`, New opens a modal card instead of adding an inline row.

### Record Entry (insert lifecycle) (ABSOLUTE RULE)
Matches classic ERP (see `screenshots/GeneralJournal01-07.png`).
- **Row created** — pre-populated with defaults, `_isNew`, not persisted, not counted in the record count.
- **INSERT** — on a confirmed cell, as soon as `shouldInsertNewRecord()` holds: the row has a user edit and every required primary key field has a value (optional PK fields may be blank but must be defined). This fires **while the cursor is still on the row**, not on row-leave. On success `_isNew`/`_pristine` are removed and `_key` holds the persisted key.
- **MODIFY** — every later confirmed cell on that row.
- **DISCARD** — leaving a new row with no user edits removes it client-side with no API call (`isEmptyNewRecord` → `cleanupEmptyNewRows`, or the row filter in `confirmAndMoveTo`).
- Never test "has any non-empty field" to decide an insert — defaults would insert the row the moment it is created. Always compare against `_pristine`.
- A user-edited new row whose required PK fields are still blank stays uncommitted; it inserts on the first confirmed cell after they are filled. An insert that fails keeps the row uncommitted and editable; the next confirmed cell retries it.
- Insert timing depends on **what the user changed**, never on where focus went. **Do NOT use `document.activeElement`** to detect row changes — it's unreliable when async validation re-renders mid-await, destroying the input and moving focus to `document.body`.
- The `required` flag is sent from the backend via table YAML metadata → `TableMetadata` → page field definitions.
- **DelayedInsert pages** (`delayed_insert: true` in the list page YAML, DelayedInsert): the INSERT waits until the user **leaves the row** (`handleCellBlur(..., leavingRow = true)`: moving to another row, Tab/Enter past the last row, focus leaving the table). Use it where the user types a composite key, e.g. User Members (`user_id + role_id + company`): inserting on the first field would save a half-entered key, and a blank `company` means "all companies". Escape does **not** insert (it cancels). Without the flag, a list inserts on the first validated field (rule above).
- **Renaming keys**: changing a primary key field of a saved record is a MODIFY that **renames** the key: the generated `Modify()` SETs changed PK fields and matches the old key in its WHERE clause (Rename), then carries the new value over to every field of another table whose `table_relation` points to that key (`renameReferences`, generated from the YAML relations by tablegen; for a global table in every company). The API runs the modify in a transaction (`modifyInTransaction`, generated `SetDB`), so the rename and its cascade commit or roll back together. The frontend addresses an existing row by `_key` (its persisted key), not its current field values, so PK edits reach the right record.
- **Duplicate check**: `InsertRecord` checks for an existing record with the **full** primary key (`fullPrimaryKey`); `GetPrimaryKeyValue()` returns only the first key field and must not be used to identify composite-key records.
- After a successful insert/modify the page header shows the "Saving…"/"✓ Saved" indicator (`saveState`), the only feedback that a row committed.

### Cross-field validation (OnValidate auto-fill)
- For a new row, confirming a field the user changed calls `api.validateField(table, field, value, record)` with the **full in-progress record**. The backend hydrates the table from it, runs `ValidateField` (→ the table's `OnValidate_<Field>` triggers), and returns the resulting record, which is merged into the row. A trigger can therefore fill sibling fields (e.g. Account No. → Account Name).
- Existing rows get trigger results from the `modifyRecord` response, which is merged into the row the same way.
- `InsertRecord`/`ModifyRecord` only VALIDATE fields whose value changed, in table field order (`validateChangedFields`), so a stale sibling in the payload never overwrites a value another field's trigger filled in.
- Triggers reach the API through the registered logic of the table (`Register<Table>Logic`, see "Table business logic"): the generated `Insert`/`Modify`/`Delete`/`ValidateField` call it for every record, however it was created — there is no per-record wiring to forget.

### Cell Value Auto-Save
- When a cell value is "confirmed" (see keyboard tables above), existing records call `modifyRecord`. New records follow the Record Entry rules above.
- The `isSaving` guard must be set **before** any async validation to prevent race conditions from concurrent save events.
- For existing rows, `table_relation` fields without an advanced lookup are checked via `api.validateField`; fields using `LookupDropdown` (advanced lookup) skip it since the component validates internally. New rows always validate the user's change (see above).
- **Only changed rows are saved**: every editable row carries `_saved` (its values as last saved, set in `toEditableRecords` and after each insert/modify). `handleCellBlur` returns at once for an existing row with no change against `_saved` — no validation, no MODIFY, no list reload. Moving through cells used to MODIFY every row passed (a write plus a reload per key, which also hit the 300 requests/minute rate limit).
- Failed saves of existing records revert to `_saved`.
- Concurrent save events are queued in order in `pendingSaves` (including the field name) and processed one by one after the current save completes. Never a single slot: a third quick edit overwrote the waiting one and was never saved.
- `PageRenderer.handleListSave` reloads the list window without awaiting it, so the next cell's save never waits for a reload.
- After an `await`, locate a new row by `_tempId` (never by a stale index) — `exitToNavigation()` can clear `editableRecords` and rows can be removed mid-save.

### Lookup Fields (ABSOLUTE RULE)
- Fields with `table_relation` that have `columns` + `rows` (advanced lookup) must render using `LookupDropdown` with `compact={true}` — providing a table-style dropdown with column headers, type-ahead filtering, and keyboard navigation.
- Fields with only `simple` lookup data render as `<select>`.
- Fields with no lookup render as plain `<input>`.
- Never use `<datalist>` for lookup fields.
- The `LookupDropdown` must be wrapped in a `<div data-row data-col>` container for focus management.

### On-demand lookups (large related tables) (ABSOLUTE RULE)
- `getLookupValues` sends a relation's rows with every list/card response only when the related table has at most `lookupInlineLimit` (200) rows. Larger tables (customers, G/L accounts, …) are sent as `{columns, lazy_url, total}` without rows: sending all 10,000 customers with each Customer Ledger Entries window cost ~0.4 s per request.
- `LookupDropdown` with `lazyUrl` loads up to 50 rows from `GET /api/tables/:table/lookup/:field` when it opens and 150 ms after typing (server search over the key and the relation's `lookup_columns`, ordered by key, `key=` for one exact key); it shows "N of M shown — type to search" when more match. Decide advanced vs simple rendering with `isAdvancedLookup()` (`utils/fieldHelpers.ts`), never by `rows.length` alone.
- Tab/Enter/blur stay synchronous: a typed key that is not among the loaded rows is taken as typed (uppercased); the server then checks it on save — `InsertRecord`/`ModifyRecord` reject a changed table-relation value that does not exist (`checkRelations`), the same check as `/validate`. Never let a relation field be saved without that check.
- Give relations to large tables `lookup_columns` (e.g. customer: `no, name, city`) so users can search by name.

### LookupDropdown Select vs Blur (ABSOLUTE RULE)
- When the user selects a value from a `LookupDropdown` (via click or Enter), the component must call `onselect` (to set the value) and re-focus its input — it must NOT call `onblur` or trigger a save.
- **Enter on a closed dropdown is never swallowed**: it commits the typed text and bubbles — in a list cell to `handleLookupCellKeyDown` (move down / new row on the last row), on a card page to the Enter-to-next-field handler. Swallowing it left the user stuck in the field. The dropdown opens with ArrowDown, Alt+ArrowDown or F4, never Enter.
- **Tab commits the typed text synchronously** (`commitTypedInput`): an exact code match (case-insensitive, so `hans` → `HANS`), else the highlighted matching row, else the first code starting with the text. No match → error and the focus stays in the field (the list's Tab handler runs right after and would otherwise save the partial text).
- The save fires only when focus actually **leaves** the component (e.g., user presses Tab or moves to another cell).
- This keeps the save tied to the user confirming the cell: a lookup pick inserts a new record (Record Entry rules) only when focus leaves the cell, never on the pick alone.
- **Re-open guard**: After `handleSelect` re-focuses the input, `onfocus` must NOT re-open the dropdown. The `selectHandled` flag prevents this — `openDropdown()` skips when `selectHandled` is true. The flag is cleared when the user types, toggles the dropdown, or presses ArrowDown to explicitly re-open.

### Focus Management
- **Row keys**: rows are keyed by `getRecordKey` — `_tempId`, then the persisted key `_key`, then the primary key. Never key an editable row by its live field values: typing a new primary key changed the key per character, Svelte recreated the row, the input lost the focus and the half-typed key was saved as a rename.
- **Leaving the table while editing**: `handleEditingInputBlur` acts only while the state is still `cell-editing` and focus is outside `.table-container` (another page, or the list's own search box/toolbar): it saves the cell and calls `exitToNavigation(false)`, which keeps the focus where the user put it. Keyboard and click moves switch to `cell-selected` first and save the cell themselves, so the blur of the removed input is ignored. The page auto-focus (`listPageElement.focus()`) only takes the focus when nothing else has it.
- `focusCell` / `focusCellSelectedElement` focus as soon as Svelte has updated the DOM (`afterRender`: `tick()`, one retry after 50 ms if the element is not there yet). A fixed 50 ms delay lost the characters typed right after the first one.
- `focusCell(rowIndex, colIndex)` handles three cell types:
  - Direct `<input>` elements (via `input[data-row][data-col]`)
  - `<select>` elements (via `select[data-row][data-col]`)
  - `LookupDropdown` wrapper `<div>` containers (via `div[data-row][data-col]`, then focuses the inner `<input>`)
- `focusCellSelectedElement(row, col)` focuses the cell-selected `<div>` via `[data-cell-row][data-cell-col]` attributes.
- **Scrolling (sticky header)**: the column header is sticky, so the browser's `scrollIntoView` / default `focus()` scrolling parks a row moving up *behind* the header. Rows are scrolled by `scrollRowIntoView` (selection) and `scrollCellIntoView` (cells, also horizontal), which keep them below the header; cell focus uses `focus({ preventScroll: true })`. Never use plain `scrollIntoView` or `focus()` on list rows/cells.
- Entering cell-editing (F2, F8, double-click, typing) places the **cursor at the end** of the text — never select-all — so a value can be amended (`PAY-TERM01` → Backspace → `PAY-TERM02`). `enterCellEditing` calls `focusCell(..., false)`; `placeCursorAtEnd` skips inputs without text selection (date/time).
- **LookupDropdown keyboard wrapper**: The `<div data-row data-col>` wrapper has an `onkeydown={handleLookupCellKeyDown}` handler that intercepts Tab/Enter/Escape/F2 after they bubble up from LookupDropdown. Keys already handled by LookupDropdown (e.g., ArrowDown when dropdown is open) are skipped via `event.defaultPrevented` check.

### Search and Sorting
- **Windowed loading (ABSOLUTE RULE)**: a list never loads the whole table. `PageRenderer` loads a **window** — the rows that fit on the page plus two pages above and below, at most 200 (`utils/listWindow.ts`) — with `offset`/`limit`, and keeps `total` (all matching records). `displayRecords` is that window (`editableActive ? editableRecords : records`); `windowOffset + index` is the position in the whole list. Moving past the window or within a page of its edge (keys, mouse wheel/scrollbar) loads the window around the target: navigation keys go through `moveToRow` → `windowIndexOf`, cell modes through `confirmAndMoveTo` (which saves the leaving cell first). Keys pressed during a load are queued (`pendingTarget`), never dropped. The window is never replaced while a user-edited, uncommitted new row exists (`hasUncommittedNewRow`). Rendering all rows was measured at ~40 s for 10,000 customers — never go back to loading everything.
- **Search**: Case-insensitive "contains" over the visible stored columns, **on the server** (`search` + `search_fields` on `/list`, generated `SetSearch`), 300 ms after typing stops; it covers all records, not only the loaded window. FlowFields are computed, not stored, so they are not searchable. New, uncommitted rows (`_isNew`) stay visible, so Alt+N works while a search is active.
- **Row indexes are displayed positions (ABSOLUTE RULE)**: `selectedIndex`, `currentCellRow`, `rowIndex`, `prevRow` etc. are positions in `displayRecords` (the loaded window, after server search and sort) — never index `records` or `editableRecords` with them. Read rows as `displayRecords[i]`; the selected saved record is `findSelectedRecord()`. Insert/remove rows in `editableRecords` by identity (`insertRowAfter`, `filter(r => r !== row)`) and map back with `displayIndexOf()`. After an `await`, update the row object itself, not `editableRecords[index]`. Mixing the index spaces made Edit/Delete act on the wrong record and cell edits land in the wrong row when the list was searched or sorted.
- **Column sorting**: Click a column caption to sort **on the server** (`sort_by` + `sort_order` asc/desc, generated `SetAscending`), over all records; again for the other direction; an arrow shows the sort. The primary key always follows the sort key in `ORDER BY`, so consecutive windows never overlap or skip rows. Sum/Count FlowFields sort too (`page.sortable_flow_fields`; the generated `SetCurrentKey` orders by `flowFieldExpr`, the same subquery as FlowField filters, its arguments after the WHERE arguments — `getOrderByClause` returns them); other FlowFields are not sortable.
- **Column header menu** (▾ on every header, `ColumnHeaderMenu.svelte`, helpers in `utils/columnHeader.ts`): Sort Ascending/Descending, Filter to This Value (the cursor row's value through `onfilter` — the filter pane stays the one source of filters; disabled for blank values and values with filter operators `| .. * < >`; booleans filter as 1/0), Clear Filter (when the column has one), Hide Column. Headers can be dragged onto another header to move the column. Hide and move write `columnCustomizations` exactly like the Customize dialog (`visible`, `order`). While the menu is open the list's global key handler leaves keys to it (Esc closes only the menu). Freeze pane was left out on purpose.

### Column Customization
- Users can hide, reorder, and resize columns via the Customize dialog.
- Customizations are persisted per user per page to localStorage.
- The `visibleColumns` derived value applies visibility and custom ordering.

### Composite Key Encoding
- `getRecordId()` joins all PK values with commas for composite keys.
- Empty string is a valid PK value (e.g., blank company = all companies access).
- The function accepts `primaryKeyFields` array for composite support.

### Boolean Fields in List
- Boolean fields render as checkboxes in both cell-selected and navigation modes.
- Detection uses `typeof record[field.source] === 'boolean' || fieldTypes[field.source] === 'bool'` — the `fieldTypes` check is essential for new rows where values are initialized as empty strings.
- In cell-selected mode, Space toggles the checkbox. Boolean fields have no cell-editing state.

### Modal Card Integration
- When `modal_card: true` on the list page, Edit and New actions open a `ModalCardPage` overlay.
- The modal auto-saves on field changes (no explicit Save button) — mirrors classic ERP behavior.
- On modal close, if any changes were made (`modalHadChanges`), the list data is refreshed.
- Save errors on new records set `modalSaveBlocked = true` to prevent further edits until cleared.
- **Card insert waits for the key (ABSOLUTE RULE)**: a new record on a card (modal `handleModalSave`, page `handleCardSave`) is INSERTed only once every primary key field has a value (`hasPrimaryKey`); earlier field changes stay on the form and go with the insert. Inserting earlier failed with "… cannot be empty" and blocked the modal — e.g. Chrome/Edge password managers filling the User card's password before the User ID. Record inputs set `autocomplete` (`new-password` for masked fields, `off` otherwise) so browsers do not fill saved logins into them.

### Codeunit Execution from List Pages
- Actions with `run_object: codeunit:ID` or `run_object: field:fieldname` execute codeunits via the job system.
- A `ProgressModal` shows real-time progress, handles confirm dialogs, input request dialogs, and error display.
- The job SSE stream delivers `progress`, `confirm`, `request_input`, and completion events.
- On completion with data (e.g., PDF), the result is passed to the `onData` callback for download handling.

## API Response Format

```
Success: { "success": true, "data": { ... }, "captions": { "table": "...", "fields": { ... }, "field_types": { ... } } }
Error:   { "success": false, "error": "..." }
List:    { "success": true, "data": { "records": [...], "total": N, "page": N, "page_size": N }, "captions": { ... } }
```

The `captions` object may include: `table` (translated table name), `fields` (field name → caption), `field_types` (field name → type like `"bool"`, `"code"`, `"text"`), `options` (enum field values), `lookups` (table relation data).

## API Client Usage

```typescript
// Generic: import { api } from '$services/api'
api.listRecords('Table', { filters, sort_by, sort_order, page, page_size })
api.getRecord('Table', id)
api.insertRecord('Table', data)
api.modifyRecord('Table', id, data)
api.deleteRecord('Table', id)
api.validateField('Table', field, value)

// Typed: import { customerApi } from '$services/api'
customerApi.list()
customerApi.get(id)
```

## Commit Convention

Uses Conventional Commits with Release Please for automated versioning:
- `feat:` new feature (minor bump)
- `fix:` bug fix (patch bump)
- `perf:` performance improvement
- `refactor:` code refactoring
- `docs:` documentation
- `ci:` CI/CD changes
- `chore:` miscellaneous (hidden from changelog)
- `BREAKING CHANGE:` in footer (major bump)

## Git Identity

Always use this identity for commits — never update git config to anything else:
- Name: `Hans J. Lachmann`
- Email: `hansjlachmann@hotmail.com`
- Never add `Co-Authored-By` lines to commit messages

## CI Pipeline

GitHub Actions runs on push/PR to main:
1. Backend: `go vet` + `golangci-lint` + tests with race detection + build
2. Frontend: type check + unit tests + build + Playwright E2E
3. Release Please on main push
4. Docker multi-arch build to ghcr.io on release
