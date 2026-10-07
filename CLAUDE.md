# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

OpenERP is a full-stack ERP system inspired by Microsoft Dynamics NAV/Business Central. Go backend (Fiber) + SvelteKit frontend + PostgreSQL/SQLite.

**Required runtimes**: Go 1.24, Node.js 22. Always target these versions.

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
cd tools/tablegen && go build -o tablegen .
cd backend/business-logic/tables
../../../tools/tablegen/tablegen -input=definitions/customer.yaml -output=.
```

### Docker
```bash
docker-compose up                    # Full stack (backend + frontend + postgres + nginx)
docker-compose -f docker-compose.prod.yml up  # Production
```

## Architecture

```
cmd/api-server/          Entry point - DB init, object registry setup, server start
backend/
  api/                   Fiber HTTP server, routes, handlers, middleware
  business-logic/        Domain layer
    tables/              Table definitions (Go structs + YAML definitions/)
    pages/               Page definitions (YAML-driven UI metadata)
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
generated/tables/        Auto-generated table constants (from tablegen)
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
- **Table registration**: When adding a new table, register it in `cmd/api-server/main.go` via `registry.RegisterTable(ID, &TableStruct{})`.
- **Company-scoped tables**: All business data tables are prefixed with the company name (e.g., `"cronus$Customer"`). System tables (User, Language) are global.
- **Database auto-detection**: If `DB_HOST` env var is set, uses PostgreSQL; otherwise SQLite with interactive prompts.
- **Opening a table in code**: always `InitWithDBType(db, company, dbType)` with the session's/handler's database type — there is deliberately no `Init(db, company)`: the type decides the SQL placeholders, and the old SQLite default silently broke code on PostgreSQL (production).
- **Table sync**: Additive schema changes (new tables/columns) happen automatically on startup. Destructive changes require migrations.
- **Migrations**: Sequential versioned files in `backend/business-logic/migrations/`. Auto-registered via `init()`. Use `ctx.ForEachCompanyTable()` to apply across all companies. Supports distributed locking for Kubernetes.
- **Extensions**: Custom tables/pages/codeunits use IDs 50,000-99,999. Core uses 1-49,999. Extensions are `.extend.yaml` files merged at build time via `extmerge`.
- **YAML-driven definitions**: All table definitions, list pages, and card pages are defined in backend YAML files. Never hardcode table structures or page layouts in Go or Svelte code.
- **Generic pages**: All frontend pages (list and card) must be fully generic — driven by page metadata from the backend, not hardcoded per table. Never create table-specific page components.
- **Menu assignment**: The main menu is determined per user via `User.menu` → `Menu.code` (FK). The Menu table defines a `filename` pointing to a YAML menu definition file in `backend/business-logic/pages/menus/`. The `filename` must include the `.yaml` extension (e.g., `"jobqueue.yaml"`). Main menu items must always be defined in backend YAML files — never hardcode menu items in the frontend.
- **Frontend proxy**: Vite dev server proxies `/api` to the Go backend at `localhost:8080`.
- **Per-request JWT sessions**: Authentication uses JWT tokens in HTTP-only cookies (`openerp_session`). The auth middleware (`backend/api/middleware/auth.go`) parses the JWT on every request, loads/creates a session, and stores it in `c.Locals("session")`. Handlers access the session via `getSession(c)` (in `handlers/helpers.go`), never via the old `session.GetCurrent()` global. Sessions are cached in-memory (`SessionCache`) keyed by `userID:company`. Login/logout/company-switch re-issue the JWT and update the cache. The `JWT_SECRET` env var must be set in production; in dev a random key is generated (sessions won't survive restarts). `SecureCookie` is automatically enabled when `DB_HOST` is set (production). The `MenuBar` component is mounted once in the root layout and does NOT re-mount on client-side navigation — use `$effect` watching stores to reload data after login/logout.
- **Migration ordering**: Migrations run BEFORE table sync in `EnterCompany`. If a migration needs to INSERT into a table, it must first ensure the table exists with `CREATE TABLE IF NOT EXISTS`. Use idempotent inserts (`ON CONFLICT DO NOTHING` for PostgreSQL, `INSERT OR IGNORE` for SQLite).
- **Case-sensitive DB comparisons**: PostgreSQL string comparison is case-sensitive. Always use the canonical (uppercase) user ID from the database (e.g., `user.User_id.String()`) for permission and lookup queries — never use raw user input directly.
- **Composite primary keys**: Tables can have composite primary keys (e.g., `User_Member` has `user_id + role_id + company`). `TableMetadata` stores `map[string][]string` for all PK fields. The API uses comma-separated PK values in URLs (e.g., `/delete/HANS2,READER,TEST-COMPANY`), parsed by `parseRecordKey()` in the backend.
- **Field type metadata**: The backend sends `field_types` (a `map[field_name]type_string`) in API captions for both page definitions and table record responses. The page endpoint (`/api/pages/:id`) reads types from YAML table definitions via `TableMetadata.GetFieldType()` (returns lowercase YAML types like `"bool"`, `"code"`, `"text"`). The table endpoints (`/api/tables/...`) use the Go `FieldType` constants (returns `"Boolean"`, `"Code"`, `"Text"`). The frontend uses the page endpoint's `field_types` for rendering decisions (e.g., boolean checkbox detection).
- **SIFT (Sum Index Fields, BC/NAV)**: a table key can declare `sum_index_fields` in its YAML (e.g. Customer Ledger Entry key `customer_open: [customer_no, open]` sums `remaining_amt_lcy, sales_lcy, amount_lcy`). For each such key the database keeps a totals table `company$Table$SIFT$key` (key columns as primary key, one column per sum field, `cnt`), maintained by **database triggers** generated by `backend/foundation/sift` (Postgres PL/pgSQL + SQLite triggers) in the same statement/transaction as the entry. Rules:
  - Triggers only apply differences (`sum = sum + x`, `ON CONFLICT DO UPDATE`): never read a total and write it back. Totals rows with `cnt = 0` are deleted. NULL key values are stored as blank (`''`, 0, false, blank date).
  - FlowFields (card `CalcFields`, list `CalcFieldsForRecords`, Sum and Count) automatically read the totals when a SIFT key of the source table covers all flow filter fields and the summed field (tablegen `resolveFlowFieldSIFT`); otherwise they add up the entries and tablegen prints a warning if the source table has SIFT keys.
  - Startup sync (`SyncKeys`, called by `CreateTableWithDBType` and table sync) builds/rebuilds totals when the key definition's fingerprint changed (fields added/removed), when the totals table is missing, and drops totals of removed keys; definitions are stored in the global `_sift_definition`. Each rebuild is one transaction; on Postgres an advisory lock lets only one instance build. Deleting a company calls `sift.DropCompany`.
  - Object names over 63 bytes are shortened with a hash (`sift.ObjectName`) — never build SIFT object names by hand.
  - Tests: `go test ./backend/foundation/sift/` runs on SQLite; set `TEST_POSTGRES_DSN=postgres://openerp:openerp@localhost:5432/sift_test?sslmode=disable` (database `sift_test` in the local docker Postgres) to also run them on Postgres, incl. the concurrency test.
- **Verify SIFT (codeunit 50110)**: run from the Job Queue in a company; compares every SIFT totals table with a fresh `GROUP BY` over its entries (`sift.VerifyKey`, generated `VerifySIFT`). Parameter `VERIFY` (default) reports — differences fail the run, so a scheduled job goes to Error and can e-mail; `REPAIR` rebuilds the keys that differ (`sift.RebuildKey`, one transaction, entry table locked against writes on Postgres). SIFT totals rows are updated on every posting, so Postgres leaves dead row versions until autovacuum runs: right after a bulk load (e.g. HEAVY demo data) totals reads are slower until then.
- **FlowFilter fields (BC/NAV FieldClass FlowFilter)**: a table field with `flow_filter: true` (types `types.Date`, `bool`, `int`, `types.Text`, `types.Code`) is not stored; it holds a filter expression the user sets, e.g. Customer `date_filter`. A FlowField applies it with a flow filter of type `filter`: `{field: posting_date, type: filter, value: date_filter}` — only when the FlowFilter has a value. Rules:
  - The generated record keeps each FlowFilter's expression; set it with `SetFlowFilter(field, expr)` (validated by `backend/foundation/flowfilter`, which builds the SQL with bind parameters: `a..b`, `..b`, `a..`, `|`, `<>`, `*` for Text/Code, Yes/No). The API takes `flow_filters` (JSON `[{field, expression}]`) on `/list` and `/card`; page definitions send `flow_filter_fields` (name, kind, caption).
  - Dates reach the backend as ISO only. The user types BC syntax in the filter pane's "Filter totals by" section (`01.01.26..31.03.26`, `t` = today); `utils/flowFilter.ts` converts it (`toApiFlowFilter`, `apiFlowFilters`): dotted dates are always day.month[.year] (`01.04` = this year) whatever the UI language — dots only occur in day-first formats, and an en-US user typing `01.04.26` means 1 April; slash/dash dates follow the UI language's order; ISO always works. The empty field shows the user's format as a hint. The modal card opened from a list gets the list's FlowFilters.
  - SIFT: tablegen resolves two keys per FlowField — without and with the FlowFilter fields — and the generated code picks at run time; with no key covering the FlowFilter fields a filtered FlowField sums the entries (tablegen warns). Customer has no date SIFT key on purpose: with ~1–2 entries per customer and day a (customer, posting_date, open) key holds about as many rows as entries and gave no speed-up (measured on HEAVY demo data), while costing a totals update per posting. Add one only for tables with many entries per day.
- **Sensitive fields (ABSOLUTE RULE)**: a table field with `sensitive: true` in its YAML (e.g. `User.password_hash`) never leaves the server through the table API. Handlers send records with `ftables.PublicMap(table)`, never `table.ToMap()` (that is for internal use: keys, comparisons); field names from the client in filters, `sort_by` and `search_fields` are checked with `ftables.IsQueryableColumn` (a filter or search on a hash reveals it piece by piece); insert/modify/validate reject a payload that sets a sensitive field (`rejectSensitiveFields`). Tablegen refuses a sensitive primary key/FlowField and a table relation whose dropdown shows a sensitive field. User passwords are set through the virtual `password` field (hashed by `SetPassword`). Mark every new secret or hash field `sensitive: true`, and never put one on a page.
- **Masked fields (ABSOLUTE RULE, BC ExtendedDatatype Masked)**: a secret the user must be able to set but never read back (e.g. `SMTP_Setup.password`) is `masked: true` in the table YAML (Text/Code only). `PublicMap` sends `ftables.MaskedValue` (`••••••••`) when it has a value, "" when empty; writes go through `DropMaskedPlaceholders`, so the placeholder sent back keeps the stored value and "" clears it; like sensitive fields it cannot be filtered, sorted or searched. Server code (e.g. the mailer) reads the real value from the table record. The page metadata marks masked fields (`field.masked`, from the table YAML or a page field's `masked: true` — e.g. the User card's virtual `password`, hashed by the API) and the frontend renders them as password inputs on cards, list cells and modal cards — decide by `field.masked`, never by the field name. Use `sensitive` for values nobody may set or see (hashes), `masked` for secrets an admin enters.
- **Keys inside panels**: list shortcuts (`use:shortcuts` on the list page) skip keys from inside an element marked `data-own-keys` (the filter pane). Svelte 5 delegates `onkeydown` to the app root, so a `stopPropagation` in a component's handler runs after the list's native shortcut listener — it cannot stop it.
- **Company display name**: `Company.name` is the technical key — it prefixes every company table, so it allows only `a-z 0-9 _` (lowercased on create). `Company.display_name` (BC/NAV Display Name, e.g. "Demo Company 01") is what users see. `GET /api/auth/companies` returns `{name, display_name}` objects; the session, JWT and `POST /api/auth/company` use `name`. In the frontend show companies via `companyLabel`/`companyLabelFor` (`utils/company.ts`), never the raw name.
- **Renaming / deleting a company (BC/NAV Rename)**: changing `Company.name` (Companies page) moves the company's data. The generated `Modify` runs the wrapper's `OnRename` trigger whenever a primary key field changes (every table; `OldValue(field)` gives the xRec value). `Company.OnRename` validates the new name (`company.NormalizeName`: lower case `a-z 0-9 _ -`, 2–50), refuses an existing company, then `company.RenameObjects` drops the old SIFT objects (their trigger functions name the totals tables) and renames every `old$…` table, index and sequence, and `syncCompanyKeys` rebuilds indexes and SIFT totals for the new name; the generated cascade updates User Member. All of it runs in the API's modify transaction (`RenameObjects` refuses a plain `*sql.DB`) — a failure at any step rolls everything back. Deleting a company drops its tables and SIFT objects in the delete's transaction (`company.DropObjects`). Find a company's objects only by exact name prefix (`company.Tables`), never `LIKE 'name$%'` — `_` is a LIKE wildcard. Afterwards `AuthHandler.CompanyChanged` drops the company's cached sessions and re-issues the renaming user's cookie; any other token naming a company that no longer exists is treated as logged out by the auth middleware. At startup a missing `COMPANY_NAME` company is only created on an empty database; otherwise an existing company is entered.
- **Index names**: a company table's index is `company$Table$key`; a global table's index is `Table$key`, shared by all companies (the old company prefix created one copy per company — removed by migration 007).
- **Demo data**: codeunit 50100 (`codeunits/demodata_create.go` → package `business-logic/demodata`) fills the current company from `demodata/data/*.yaml`, run from the Job Queue (Run or scheduler); the job's `parameter` is `SMALL` (default) or `LARGE`. One transaction, fixed random seed, refuses a company that already has customers. Records go through `Insert(true)`, so table triggers apply (Customer `OnInsert` sets Status Open — Blocked is applied with a Modify afterwards). When a new business table gets demo data, add it to the YAML and `demodata.Create` in the same change.
- **FlowFields (BC/NAV style)**: Tables can define computed aggregate fields from related tables. Defined in table YAML with `flow_field: true`, `calc_formula` (Sum, Count), `source_table`, `source_field`, and `flow_filters`. The tablegen tool generates `CalcFields()` methods. The API automatically calls `CalcFields()` for list and get endpoints when flow fields are in the requested fields. Flow fields are read-only and not persisted to the database. Example: `Customer.balance_lcy` sums `CustomerLedgerEntry.remaining_amt_lcy`, `Job_Queue.number_of_entries` counts related `Job_Queue_Entry` records. Lists must not call `CalcFields` per row (one query per row and FlowField — 28 s for 10,000 customers): `ListRecords` reads the rows, then calls the generated `CalcFieldsForRecords(records, fields...)`, which runs one grouped query per FlowField (IN list for ≤500 keys, whole source table above) and only for the requested fields. Keep that pattern for any endpoint returning many records.

## Page Definition Guide

All pages are defined in backend YAML files at `backend/business-logic/pages/definitions/`. File naming convention: `{ID}-{name}.yaml`. Pages auto-load from this directory — no Go registration needed.

### List Page YAML Structure
```yaml
page:
  id: 22                      # Unique page ID (core: 1-49999, extensions: 50000-99999)
  type: List                   # Must be "List"
  name: Customer List          # Internal name
  source_table: Customer       # Table to read from (must be registered in main.go)
  caption: Customer List       # Display title (translated via pages.yaml translation files)
  card_page_id: 21             # (optional) Associated card page for row click / Edit action
  modal_card: true             # (optional) Open card in modal instead of navigating to card page
  editable: true               # (optional) Enable inline editing in the repeater
  delayed_insert: true         # (optional) BC DelayedInsert: insert a new row only when leaving the row

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
      run_object: codeunit:50010        # (optional) Run codeunit by ID
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
      run_object: codeunit:50010
```

### Key Rules
- **Card actions**: only `promoted: true` actions are shown as buttons on a card (there is no overflow menu); others are reachable by their shortcut only. `run_page_filter_field` / `run_page_filter_value` open `run_page` filtered to the current record (like list drilldowns).
- **Table names**: pages use registry names (`Customer_ledger_entry`), table YAML files display names (`Customer Ledger Entry`). Table metadata (`TableMetadata`, primary key fields, field types) normalizes both (`metadataKey`); a page whose source table finds no metadata gets no `primary_key_fields`, and its rows are keyed randomly (focus lost on every render).
- **Lookup field definitions**: `table_relation` with `lookup_columns` is defined in the **table** YAML (`backend/business-logic/tables/definitions/`), not the page YAML. The page YAML only needs `table_relation: TableName` on the card field. The backend resolves lookup column data automatically and sends it in the `captions.lookups` response.
- **Field captions**: Come from `translations/{lang}/tables.yaml` by default. The page YAML `caption` field overrides the translation for a specific page only.
- **Page captions**: Come from `translations/{lang}/pages.yaml`. The YAML `caption` is the fallback if no translation exists.
- **Action captions**: Come from `translations/{lang}/common.yaml` under `common.actions.{action_name}`. The YAML `caption` is the fallback.
- **Frontend routing**: `PageRenderer.svelte` checks `page.type` and renders `<ListPage>` or `<CardPage>` — never create page-type-specific routes or components.
- **Modal card**: When `modal_card: true` on a list page, the Edit/New actions open a `<ModalCardPage>` overlay instead of navigating to a separate card page. The modal auto-saves on field changes and refreshes the list on close.
- **Drilldown fields**: List page fields can have `drilldown` (target page ID), `drilldown_filter_field` (field on target table), and `drilldown_filter_value` (field on current record). Clicking the cell navigates via `window.location.href` to `/pages/{drilldown}?filter={filter_field}={value}&back=…`. **Return (BC)**: drilldowns and card `run_page` actions add `back=<origin page with select=<record key>>` (`withReturn` in `utils/returnUrl.ts`, validated by `safeReturnUrl`). A list opened with `back` shows a close (×) button and Esc in navigation mode returns there instead of to the main menu; the origin list then opens on the window around the selected record and selects it (`initialSelect`, located via `/ids` when the list is unfiltered). Use `window.location.href` (not `goto()`) because `PageRenderer` uses `onMount` — client-side navigation won't remount the component.
- **URL filter parameter**: List pages accept `?filter=field=expression` query parameter to pre-filter on load. Parsed in `PageRenderer.svelte` into `currentFilters` and passed to the API. Used by drilldown links to show related records.

## Frontend Conventions

- **TypeScript strict mode** with `checkJs` enabled
- **Suppressed Svelte warnings**: `a11y_no_noninteractive_tabindex`, `a11y_no_noninteractive_element_interactions` (intentional tabindex patterns), `css_unused_selector` (Tailwind `@apply`)
- **Business Central color palette**: colors match BC light and dark mode (reference: `docs/BC-Screenshot-LightMode.png`, `docs/BC-Screenshot-DarkMode.png`). `tailwind.config.js` retunes Tailwind's `gray` scale to BC neutrals (light page `#ffffff`, text `#212121`, secondary `#505c6d`; dark page `gray-900` `#121212`, text `gray-50` `#f7f7f7`, secondary `gray-400` `#a4b0c4`, separators `gray-700` `#303032`) and maps `blue`/`primary` to BC teal (links `600` `#008489` light / `400` `#37a1a5` dark, tiles `500` `#00838f`). `nav-blue` is the BC app bar `#282828` in both modes. Use these classes — never introduce other hues (no indigo/purple/Tailwind blue) except red/green/yellow for status. One exception: selected-text highlight in list cells (`.cell-selected-text`, `.edit-cell-input::selection`) uses BC's selection colors — `#0078d4` with white text in light mode, `#505c6d` with `#f7f7f7` in dark mode. Raw colors in `<style>` blocks must use the same palette values. Font: Segoe UI. Dark mode via `class` strategy.
- **Svelte 5 runes**: Always use runes (`$state`, `$derived`, `$effect`, `$props`, `$bindable`) — never legacy Svelte 4 patterns (`export let`, `$:`, `$store` syntax).
- **i18n**: All labels and field captions in the frontend must come from backend translation files — never hardcode display text in Svelte components. In backend codeunits, use `i18n.GetInstance().Message(key, CurrentLanguage())` for user-facing strings (dialog titles, field labels, messages). Never hardcode English strings that are shown to the user. Never use `t(...)` as a `$props()` default or a `$state()` initial value: those are evaluated once at mount, possibly before translations have loaded, and then show the raw key (e.g. `BTN_CANCEL`) — look the default up in the template instead (`{cancelText ?? t(BTN.CANCEL)}`).
- **BC/NAV keyboard shortcuts**: Alt+N new (as in the BC web client — never Ctrl+N, Ctrl+T or Ctrl+W: browsers reserve them and never pass them to the page), Ctrl+E edit, Ctrl+D delete, Ctrl+S save, Ctrl+F find, F5 refresh, Escape cancel, Ctrl+Home/End first/last, PageUp/Down prev/next, Ctrl+O switch company (NAV Classic; a global capture-phase handler in `MenuBar` opens `CompanySwitchDialog` on every page, even inside a list cell, and blocks the browser's Open File dialog — browsers do let the page take Ctrl+O). Switching company (dialog or user menu) always lands on the main menu (`/`, full load), never reloads the current page. While the dialog is open, `companySwitchOpen` is true and window-level page key handlers (ListPage) must ignore keys
- **Keyboard Shortcuts help**: the user menu's "Keyboard Shortcuts" item opens `/help/shortcuts` as a detached window (`window.open` with the fixed name `openerp-shortcuts`, so it is reused). The root layout renders `/help/*` routes without MenuBar/Breadcrumb; Esc closes the window. Its content is `shortcutHelpSections` in `utils/shortcutHelp.ts` (translation keys `HELP_*` in `messages.yaml`, checked for every language by `shortcutHelp.test.ts`). When adding or changing a shortcut, update that list too.
- **Code field behavior (ABSOLUTE RULE)**: Fields with `types.Code` must allow typing in any case, then auto-uppercase the value on blur (when the field loses focus). This matches standard NAV/BC behavior. Never force uppercase while typing — only convert on exit. Apply this everywhere Code fields are rendered: login forms, list page cell-editing, card page fields, modal dialogs, and any other input bound to a Code field.
- **Boolean field rendering (ABSOLUTE RULE)**: Fields with YAML `type: bool` must always render as checkboxes — on card pages, list pages (edit and read-only mode), and modal card pages. Detection uses two complementary checks: `typeof value === 'boolean'` (works for existing records) OR `fieldTypes[field.source] === 'bool'` (works for new/empty records where value is `undefined`). The `fieldTypes` metadata flows from backend YAML → `TableMetadata` → page API response `captions.field_types` → `PageRenderer` → `CardPage`/`ListPage`/`ModalCardPage` → `FieldRenderer`. Never rely solely on `typeof` — new records have no value yet, so the backend metadata is essential. This is fully generic: any table field with `type: bool` in its YAML definition automatically gets checkbox rendering everywhere. On editable list pages, boolean checkboxes must be **clickable** in both cell-selected mode and navigation/read-only mode — clicking toggles the value and saves immediately. In cell-selected mode, use `handleCellBlur` (editableActive is true). In navigation mode, call `api.modifyRecord` directly (editableActive is false, so `handleCellBlur` would return early). On error, revert the checkbox value. On non-editable list pages, boolean checkboxes remain `disabled`. Never render boolean checkboxes with `disabled` on editable list pages.

- **Date formatting (ABSOLUTE RULE)**: All Date and DateTime values must be displayed using locale-aware formatting via `Intl.DateTimeFormat` with the user's session language. Date fields use `formatDate()` and DateTime fields use `formatDateTime()` from `fieldHelpers.ts`. Detection uses `isDateType(fieldTypes[field])` and `isDateTimeType(fieldTypes[field])` which handle both YAML types (`"types.Date"`) and Go FieldType constants (`"Date"`). The API wire format is always ISO (`YYYY-MM-DD` / RFC3339) — locale formatting is display-only. For card page editing, use `<input type="date">` / `<input type="datetime-local">` (browser handles locale). For ProgressModal date dialogs, use text input with locale placeholder from `getDateFormatPattern()` and convert back to ISO via `parseLocaleDate()` on submit.

- **Copy/paste in list pages (ABSOLUTE RULE)**: Ctrl+C in cell-selected mode copies the cell's display value to the system clipboard via `navigator.clipboard.writeText()`. Ctrl+V in cell-selected mode pastes from clipboard, sets the cell value, and enters cell-editing mode. In cell-editing mode, the browser handles Ctrl+C/V natively on the input element — never intercept these keys in cell-editing mode. Boolean and non-editable fields ignore Ctrl+V.

- **Option field rendering (ABSOLUTE RULE)**: Fields with YAML `type: Option` must always render using the `OptionDropdown` component — on card pages (via `FieldRenderer`), list pages (cell-editing mode), and modal card pages. Never use native `<select>` elements for Option fields. The `OptionDropdown` component provides a custom dropdown with keyboard navigation: Alt+ArrowDown/ArrowDown opens the dropdown, ArrowUp/ArrowDown navigates options, Enter selects when dropdown is open, Escape closes, F4 toggles, Space opens when closed. **Critical**: Enter must only `preventDefault` when the dropdown is open (to select). When the dropdown is closed, Enter must NOT be handled — it must bubble up to `handleLookupCellKeyDown` for cell navigation (move to next row). Never re-open the dropdown on Enter when closed. In list page cell-selected mode, Option fields show the formatted value with a `▼` dropdown arrow button (same visual pattern as lookup fields). In list page navigation/read-only mode, Option fields display as plain text (the dropdown is available when entering cell-editing mode). The `options` metadata flows from backend YAML → `TableMetadata` → page API response `captions.options` → `PageRenderer` → components. In list page cell-editing mode, `OptionDropdown` is wrapped in a `<div data-row data-col>` container using `handleLookupCellKeyDown` for Tab/Enter/Escape/F2 cell navigation (same pattern as `LookupDropdown`). `focusCell()` finds the OptionDropdown via `[role="combobox"]` selector inside the container div.

- **Enter moves to the next field on card pages (ABSOLUTE RULE, NAV/BC)**: Enter in a card field acts like Tab — focus moves to the next field in Tab order (text selected), Shift+Enter to the previous one, and leaving the field saves it exactly as Tab does. On the last field Enter stays put. Implemented once in `handleFieldEnterKey` (`utils/fieldNavigation.ts`) on CardPage's `.sections-container`, so it covers every card page and modal card. Controls that use Enter themselves (an open Option/Lookup dropdown selecting a row, a lookup rejecting typed text) `preventDefault` it and the handler leaves it alone — keep that contract when adding field controls. Textareas keep Enter for new lines.

## Generic List Page Behaviors (ABSOLUTE RULES)

All list page behaviors are driven by page metadata — never add table-specific logic in `ListPage.svelte` or `PageRenderer.svelte`.

### Three Cell States (Spreadsheet Model)

The list page uses a spreadsheet-style 3-state cell model (like Excel/LibreOffice Calc), not a simple 2-mode toggle.

1. **Navigation mode** (default) — No cell is focused. Whole rows are selected/highlighted. Arrow keys move row selection. This is the default state when the page loads.
2. **Cell-selected** — A single cell has a visible selection indicator (e.g., blue border) but the user is NOT typing in it. The cell content is displayed but not editable. Arrow keys move the selection to adjacent cells. The cursor is not visible.
3. **Cell-editing** — The cursor is inside the cell and the user is actively typing. Arrow keys move the cursor within the text (unless at a boundary). The cell contains a live input element.

#### State Transitions
- **Navigation → Cell-selected**: Click a cell, or press Enter/F2/Ctrl+E (selects first editable cell of current row).
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
| Ctrl+D | Delete selected record |
| F5 | Refresh list data |
| Escape | Navigate to home (`/`) |

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
| F8 | Copy value from the cell directly above, then enter cell-editing with the cursor at the end (NAV/BC standard). Escape reverts to the value before F8 |
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
- New rows are created by `createNewRecord()`: all repeater fields start as `""` (so composite PK fields are always defined), then the row is filled with the table's defaults from `POST /api/tables/:table/init` (`InitRecord()`, BC/NAV `OnNewRecord`). If the init call fails the row stays blank — data entry is never blocked. Values the user typed before the defaults arrived are never overwritten.
- The values a row was initialized with are kept in `_pristine`. Only changes away from them count as user edits (`hasUserEdits`) — init-supplied defaults never do.
- Only one untouched new row can exist at a time — clicking New again focuses the existing one.
- New rows are marked with `_isNew: true` and a `_tempId` for stable keyed rendering.
- Editable lists render a **trailing blank row** after the last record (only when the loaded window reaches the end of the list); clicking it starts a new record. ArrowDown/Enter past the last record of the whole list does the same — "last" is decided by `isLastRow()` (window end and no rows beyond it per `total`), never by `displayRecords.length - 1` alone.
- If `card_page_id` is set with `modal_card: true`, New opens a modal card instead of adding an inline row.

### Record Entry (BC/NAV insert lifecycle) (ABSOLUTE RULE)
Matches Business Central (see `screenshots/GeneralJournal01-07.png`).
- **Row created** — pre-populated with defaults, `_isNew`, not persisted, not counted in the record count.
- **INSERT** — on a confirmed cell, as soon as `shouldInsertNewRecord()` holds: the row has a user edit and every required primary key field has a value (optional PK fields may be blank but must be defined). This fires **while the cursor is still on the row**, not on row-leave. On success `_isNew`/`_pristine` are removed and `_key` holds the persisted key.
- **MODIFY** — every later confirmed cell on that row.
- **DISCARD** — leaving a new row with no user edits removes it client-side with no API call (`isEmptyNewRecord` → `cleanupEmptyNewRows`, or the row filter in `confirmAndMoveTo`).
- Never test "has any non-empty field" to decide an insert — defaults would insert the row the moment it is created. Always compare against `_pristine`.
- A user-edited new row whose required PK fields are still blank stays uncommitted; it inserts on the first confirmed cell after they are filled. An insert that fails keeps the row uncommitted and editable; the next confirmed cell retries it.
- Insert timing depends on **what the user changed**, never on where focus went. **Do NOT use `document.activeElement`** to detect row changes — it's unreliable when async validation re-renders mid-await, destroying the input and moving focus to `document.body`.
- The `required` flag is sent from the backend via table YAML metadata → `TableMetadata` → page field definitions.
- **DelayedInsert pages** (`delayed_insert: true` in the list page YAML, BC's DelayedInsert): the INSERT waits until the user **leaves the row** (`handleCellBlur(..., leavingRow = true)`: moving to another row, Tab/Enter past the last row, focus leaving the table). Use it where the user types a composite key, e.g. User Members (`user_id + role_id + company`): inserting on the first field would save a half-entered key, and a blank `company` means "all companies". Escape does **not** insert (it cancels). Without the flag, a list inserts on the first validated field (rule above).
- **Renaming keys**: changing a primary key field of a saved record is a MODIFY that **renames** the key: the generated `Modify()` SETs changed PK fields and matches the old key in its WHERE clause (BC/NAV Rename), then carries the new value over to every field of another table whose `table_relation` points to that key (`renameReferences`, generated from the YAML relations by tablegen; for a global table in every company). The API runs the modify in a transaction (`modifyInTransaction`, generated `SetDB`), so the rename and its cascade commit or roll back together. The frontend addresses an existing row by `_key` (its persisted key), not its current field values, so PK edits reach the right record.
- **Duplicate check**: `InsertRecord` checks for an existing record with the **full** primary key (`fullPrimaryKey`); `GetPrimaryKeyValue()` returns only the first key field and must not be used to identify composite-key records.
- After a successful insert/modify the page header shows the "Saving…"/"✓ Saved" indicator (`saveState`), the only feedback that a row committed.

### Cross-field validation (OnValidate auto-fill)
- For a new row, confirming a field the user changed calls `api.validateField(table, field, value, record)` with the **full in-progress record**. The backend hydrates the table from it, runs `ValidateField` (→ the wrapper's `OnValidate_<Field>()`), and returns the resulting record, which is merged into the row. A trigger can therefore fill sibling fields (e.g. Account No. → Account Name).
- Existing rows get trigger results from the `modifyRecord` response, which is merged into the row the same way.
- `InsertRecord`/`ModifyRecord` only VALIDATE fields whose value changed, in table field order (`validateChangedFields`), so a stale sibling in the payload never overwrites a value another field's trigger filled in.
- Wrapper triggers reach the API because each wrapper overrides `InitWithDBType` to call `SetTriggers(...)` and `SetSelf(t)`; the generated `ValidateField` dispatches to wrapper `OnValidate_*` overrides through `self`. When adding a wrapper by hand, keep that override — without it OnInsert/OnModify/OnDelete and OnValidate overrides silently never run from the API.

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
- **Column sorting**: Click column headers to sort **on the server** (`sort_by` + `sort_order` asc/desc, generated `SetAscending`), over all records. Toggle asc/desc on the same column. The primary key always follows the sort key in `ORDER BY`, so consecutive windows never overlap or skip rows. FlowField columns (`page.flow_fields`) have no sort button.

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
- The modal auto-saves on field changes (no explicit Save button) — mirrors Business Central behavior.
- On modal close, if any changes were made (`modalHadChanges`), the list data is refreshed.
- Save errors on new records set `modalSaveBlocked = true` to prevent further edits until cleared.

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
