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
- **FlowFields (BC/NAV style)**: Tables can define computed aggregate fields from related tables. Defined in table YAML with `flow_field: true`, `calc_formula` (Sum, Count), `source_table`, `source_field`, and `flow_filters`. The tablegen tool generates `CalcFields()` methods. The API automatically calls `CalcFields()` for list and get endpoints when flow fields are in the requested fields. Flow fields are read-only and not persisted to the database. Example: `Customer.balance_lcy` sums `CustomerLedgerEntry.remaining_amt_lcy`, `Job_Queue.number_of_entries` counts related `Job_Queue_Entry` records.

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
    - name: Generate Ledger Entries
      caption: Generate Ledger Entries
      promoted: true
      run_object: codeunit:50010
```

### Key Rules
- **Lookup field definitions**: `table_relation` with `lookup_columns` is defined in the **table** YAML (`backend/business-logic/tables/definitions/`), not the page YAML. The page YAML only needs `table_relation: TableName` on the card field. The backend resolves lookup column data automatically and sends it in the `captions.lookups` response.
- **Field captions**: Come from `translations/{lang}/tables.yaml` by default. The page YAML `caption` field overrides the translation for a specific page only.
- **Page captions**: Come from `translations/{lang}/pages.yaml`. The YAML `caption` is the fallback if no translation exists.
- **Action captions**: Come from `translations/{lang}/common.yaml` under `common.actions.{action_name}`. The YAML `caption` is the fallback.
- **Frontend routing**: `PageRenderer.svelte` checks `page.type` and renders `<ListPage>` or `<CardPage>` — never create page-type-specific routes or components.
- **Modal card**: When `modal_card: true` on a list page, the Edit/New actions open a `<ModalCardPage>` overlay instead of navigating to a separate card page. The modal auto-saves on field changes and refreshes the list on close.
- **Drilldown fields**: List page fields can have `drilldown` (target page ID), `drilldown_filter_field` (field on target table), and `drilldown_filter_value` (field on current record). Clicking the cell navigates via `window.location.href` to `/pages/{drilldown}?filter={filter_field}={value}`. Use `window.location.href` (not `goto()`) because `PageRenderer` uses `onMount` — client-side navigation won't remount the component.
- **URL filter parameter**: List pages accept `?filter=field=expression` query parameter to pre-filter on load. Parsed in `PageRenderer.svelte` into `currentFilters` and passed to the API. Used by drilldown links to show related records.

## Frontend Conventions

- **TypeScript strict mode** with `checkJs` enabled
- **Suppressed Svelte warnings**: `a11y_no_noninteractive_tabindex`, `a11y_no_noninteractive_element_interactions` (intentional tabindex patterns), `css_unused_selector` (Tailwind `@apply`)
- **Tailwind with BC/NAV color scheme**: `text-nav-blue` (#002050), `text-nav-lightblue` (#4472c4), font: Segoe UI. Dark mode via `class` strategy.
- **Svelte 5 runes**: Always use runes (`$state`, `$derived`, `$effect`, `$props`, `$bindable`) — never legacy Svelte 4 patterns (`export let`, `$:`, `$store` syntax).
- **i18n**: All labels and field captions in the frontend must come from backend translation files — never hardcode display text in Svelte components. In backend codeunits, use `i18n.GetInstance().Message(key, CurrentLanguage())` for user-facing strings (dialog titles, field labels, messages). Never hardcode English strings that are shown to the user.
- **BC/NAV keyboard shortcuts**: Ctrl+N new, Ctrl+E edit, Ctrl+D delete, Ctrl+S save, Ctrl+F find, F5 refresh, Escape cancel, Ctrl+Home/End first/last, PageUp/Down prev/next
- **Code field behavior (ABSOLUTE RULE)**: Fields with `types.Code` must allow typing in any case, then auto-uppercase the value on blur (when the field loses focus). This matches standard NAV/BC behavior. Never force uppercase while typing — only convert on exit. Apply this everywhere Code fields are rendered: login forms, list page cell-editing, card page fields, modal dialogs, and any other input bound to a Code field.
- **Boolean field rendering (ABSOLUTE RULE)**: Fields with YAML `type: bool` must always render as checkboxes — on card pages, list pages (edit and read-only mode), and modal card pages. Detection uses two complementary checks: `typeof value === 'boolean'` (works for existing records) OR `fieldTypes[field.source] === 'bool'` (works for new/empty records where value is `undefined`). The `fieldTypes` metadata flows from backend YAML → `TableMetadata` → page API response `captions.field_types` → `PageRenderer` → `CardPage`/`ListPage`/`ModalCardPage` → `FieldRenderer`. Never rely solely on `typeof` — new records have no value yet, so the backend metadata is essential. This is fully generic: any table field with `type: bool` in its YAML definition automatically gets checkbox rendering everywhere.

- **Date formatting (ABSOLUTE RULE)**: All Date and DateTime values must be displayed using locale-aware formatting via `Intl.DateTimeFormat` with the user's session language. Date fields use `formatDate()` and DateTime fields use `formatDateTime()` from `fieldHelpers.ts`. Detection uses `isDateType(fieldTypes[field])` and `isDateTimeType(fieldTypes[field])` which handle both YAML types (`"types.Date"`) and Go FieldType constants (`"Date"`). The API wire format is always ISO (`YYYY-MM-DD` / RFC3339) — locale formatting is display-only. For card page editing, use `<input type="date">` / `<input type="datetime-local">` (browser handles locale). For ProgressModal date dialogs, use text input with locale placeholder from `getDateFormatPattern()` and convert back to ISO via `parseLocaleDate()` on submit.

- **Copy/paste in list pages (ABSOLUTE RULE)**: Ctrl+C in cell-selected mode copies the cell's display value to the system clipboard via `navigator.clipboard.writeText()`. Ctrl+V in cell-selected mode pastes from clipboard, sets the cell value, and enters cell-editing mode. In cell-editing mode, the browser handles Ctrl+C/V natively on the input element — never intercept these keys in cell-editing mode. Boolean and non-editable fields ignore Ctrl+V.

- **Option field rendering (ABSOLUTE RULE)**: Fields with YAML `type: Option` must always render using the `OptionDropdown` component — on card pages (via `FieldRenderer`), list pages (cell-editing mode), and modal card pages. Never use native `<select>` elements for Option fields. The `OptionDropdown` component provides a custom dropdown with keyboard navigation: Alt+ArrowDown/ArrowDown opens the dropdown, ArrowUp/ArrowDown navigates options, Enter selects when dropdown is open, Escape closes, F4 toggles, Space opens when closed. **Critical**: Enter must only `preventDefault` when the dropdown is open (to select). When the dropdown is closed, Enter must NOT be handled — it must bubble up to `handleLookupCellKeyDown` for cell navigation (move to next row). Never re-open the dropdown on Enter when closed. In list page cell-selected mode, Option fields show the formatted value with a `▼` dropdown arrow button (same visual pattern as lookup fields). In list page navigation/read-only mode, Option fields display as plain text (the dropdown is available when entering cell-editing mode). The `options` metadata flows from backend YAML → `TableMetadata` → page API response `captions.options` → `PageRenderer` → components. In list page cell-editing mode, `OptionDropdown` is wrapped in a `<div data-row data-col>` container using `handleLookupCellKeyDown` for Tab/Enter/Escape/F2 cell navigation (same pattern as `LookupDropdown`). `focusCell()` finds the OptionDropdown via `[role="combobox"]` selector inside the container div.

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
| Home / End | Select first / last row |
| Enter | If `card_page_id` set: open card page. Otherwise: enter cell-selected on first editable cell |
| F2 | Enter cell-selected on first editable cell of selected row |
| Ctrl+E | Enter cell-selected (same as F2) |
| Ctrl+N / Ctrl+Insert | Insert new row |
| Ctrl+F | Focus search input and select all text |
| Ctrl+D | Delete selected record |
| F5 | Refresh list data |
| Escape | Navigate to home (`/`) |

### Keyboard: Cell-Selected Mode

| Key | Action |
|-----|--------|
| ArrowUp / ArrowDown | Confirm value + move selection to cell above/below |
| ArrowLeft / ArrowRight | Move cell selection left/right within the row |
| Tab | Confirm value + move selection right (wraps to next row) |
| Shift+Tab | Confirm value + move selection left (wraps to previous row) |
| Enter | Confirm value + move selection down. On last data row: create new row |
| F2 | Enter cell-editing (preserve content, cursor at end) |
| Escape | Revert unsaved changes in cell, return to navigation mode |
| Delete | Clear cell content (set to empty string), stay in cell-selected |
| Backspace | Clear cell content + enter cell-editing |
| Printable character | Clear cell content + enter cell-editing with typed character |
| Ctrl+C | Copy cell value to system clipboard |
| Ctrl+V | Paste from clipboard into cell + enter cell-editing |
| F8 | Copy value from the cell directly above (NAV/BC standard) |
| Ctrl+N / Ctrl+Insert | Insert new row |
| Alt+ArrowDown (on lookup cell) | Enter cell-editing and open lookup dropdown |
| Space (on boolean cell) | Toggle checkbox value |
| Enter (on boolean cell) | Toggle checkbox value + move selection down |

### Keyboard: Cell-Editing Mode

| Key | Action |
|-----|--------|
| ArrowUp / ArrowDown | If cursor at text boundary (start/end) or all text selected: confirm + move selection. Otherwise: move cursor in text |
| ArrowLeft / ArrowRight | Move cursor within text. At text boundary: confirm + move selection to adjacent cell |
| Tab | Confirm + move selection right (wraps to next row) |
| Shift+Tab | Confirm + move selection left (wraps to previous row) |
| Enter | Confirm + move selection down. On last data row: create new row |
| F2 | Exit cell-editing → return to cell-selected (keep current value) |
| Escape | Revert cell to value before editing began, return to cell-selected |
| Ctrl+C | Native browser behavior (copies selected text) |
| Ctrl+V | Native browser behavior (pastes at cursor) |
| All other keys | Normal text input behavior |

### Key Behavioral Notes
- **"Confirm"** means: save the current cell value. For existing records this triggers `modifyRecord`. For new records this follows delayed insert rules (save only when leaving the row, not the cell).
- **Cell-selected visual**: The cell shows a distinct border/highlight (e.g., blue border) without a cursor. This must be visually distinct from cell-editing (which shows a cursor in an input).
- **Transition from navigation → cell-selected does NOT save anything** — it's purely a focus/selection change.
- **Lookup fields** (LookupDropdown, `<select>`) in cell-selected mode show the formatted value plus a **▼ dropdown arrow button**. Clicking the arrow or pressing Alt+ArrowDown enters cell-editing and opens the dropdown. The user can also enter cell-editing via F2, typing, or double-click. LookupDropdown manages its own keyboard internally (arrow keys navigate the dropdown list, Enter selects, F4 toggles, Escape closes). Cell navigation keys (Tab, Shift+Tab, Enter when dropdown closed, Escape when dropdown closed, F2) are handled by `handleLookupCellKeyDown` on the wrapper div, which intercepts events that bubble up from LookupDropdown.
- **`<select>` fields** (simple lookups) in cell-editing mode: arrow keys cycle through options natively (not intercepted by `handleCellKeyDown`). Tab/Enter handle cell navigation.
- **Boolean fields** (checkboxes) toggle on Space/Enter in cell-selected mode. They have no separate cell-editing state.

### New Record Creation
- When clicking New (or Ctrl+N), all repeater fields are initialized to `""` (empty string). This ensures composite PK fields are defined from creation.
- Only one empty new row can exist at a time — clicking New again focuses the existing empty row.
- New rows are marked with `_isNew: true` and a `_tempId` for stable keyed rendering.
- If `card_page_id` is set with `modal_card: true`, New opens a modal card instead of adding an inline row.

### Delayed Insert (NAV `DelayedInsert`)
- New records are inserted only when the user **leaves the row** — not when leaving an individual cell.
- This allows the user to fill all fields (including optional PK fields like `company`) before the insert fires.
- The insert triggers via the `forceInsert` parameter on `handleCellBlur()`. When `forceInsert` is `false` (default), new records are always deferred. Keyboard handlers pass `forceInsert: true` only when explicitly leaving the row: `confirmAndMoveTo` crossing rows, Enter/ArrowDown on last row, or focus leaving the table via `handleEditingInputBlur`.
- **Do NOT use `document.activeElement`** to detect row changes — it's unreliable when async validation (`api.validateField`) causes Svelte to re-render mid-await, destroying the input element and moving focus to `document.body`.
- Required PK fields must have non-empty values; optional PK fields (without `required: true`) can remain blank.
- The `required` flag is sent from the backend via table YAML metadata → `TableMetadata` → page field definitions.
- Empty new rows are automatically cleaned up when navigating away from them.

### Cell Value Auto-Save
- When a cell value is "confirmed" (see keyboard tables above), existing records call `modifyRecord`. New records are only saved when the user has left the row (see delayed insert above).
- The `isSaving` guard must be set **before** any async validation to prevent race conditions from concurrent save events.
- Fields using `LookupDropdown` (advanced lookup) skip server-side validation since the component validates internally.
- Failed saves revert to original values from the unmodified `records` array.
- Concurrent save events are queued via `pendingSave` and processed after the current save completes.

### Lookup Fields (ABSOLUTE RULE)
- Fields with `table_relation` that have `columns` + `rows` (advanced lookup) must render using `LookupDropdown` with `compact={true}` — providing a table-style dropdown with column headers, type-ahead filtering, and keyboard navigation.
- Fields with only `simple` lookup data render as `<select>`.
- Fields with no lookup render as plain `<input>`.
- Never use `<datalist>` for lookup fields.
- The `LookupDropdown` must be wrapped in a `<div data-row data-col>` container for focus management.

### LookupDropdown Select vs Blur (ABSOLUTE RULE)
- When the user selects a value from a `LookupDropdown` (via click or Enter), the component must call `onselect` (to set the value) and re-focus its input — it must NOT call `onblur` or trigger a save.
- The save fires only when focus actually **leaves** the component (e.g., user presses Tab or moves to another cell).
- This prevents premature inserts during data entry and is critical for delayed insert behavior on composite PK tables.
- **Re-open guard**: After `handleSelect` re-focuses the input, `onfocus` must NOT re-open the dropdown. The `selectHandled` flag prevents this — `openDropdown()` skips when `selectHandled` is true. The flag is cleared when the user types, toggles the dropdown, or presses ArrowDown to explicitly re-open.

### Focus Management
- `focusCell(rowIndex, colIndex)` uses a 50ms timeout (for Svelte DOM updates) and handles three cell types:
  - Direct `<input>` elements (via `input[data-row][data-col]`)
  - `<select>` elements (via `select[data-row][data-col]`)
  - `LookupDropdown` wrapper `<div>` containers (via `div[data-row][data-col]`, then focuses the inner `<input>`)
- `focusCellSelectedElement(row, col)` focuses the cell-selected `<div>` via `[data-cell-row][data-cell-col]` attributes.
- All inputs receive `.select()` to highlight text on focus.
- **LookupDropdown keyboard wrapper**: The `<div data-row data-col>` wrapper has an `onkeydown={handleLookupCellKeyDown}` handler that intercepts Tab/Enter/Escape/F2 after they bubble up from LookupDropdown. Keys already handled by LookupDropdown (e.g., ArrowDown when dropdown is open) are skipped via `event.defaultPrevented` check.

### Search and Sorting
- **Search**: Case-insensitive substring match across all visible columns. Filters `displayRecords` reactively.
- **Column sorting**: Click column headers to sort. Toggle asc/desc on same column. Type-aware comparison: numbers compared numerically, booleans by value, strings via `localeCompare`.

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
