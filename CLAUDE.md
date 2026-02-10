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
- **Menu assignment**: The main menu is determined per user via `User.menu` → `Menu.code` (FK). The Menu table defines a `filename` pointing to a YAML menu definition. Main menu items must always be defined in backend YAML files — never hardcode menu items in the frontend.
- **Frontend proxy**: Vite dev server proxies `/api` to the Go backend at `localhost:8080`.
- **Global session**: The backend uses a single global `session.Session` variable (not per-request). All concurrent requests share the same session state. The `MenuBar` component is mounted once in the root layout and does NOT re-mount on client-side navigation — use `$effect` watching stores to reload data after login/logout.
- **Migration ordering**: Migrations run BEFORE table sync in `EnterCompany`. If a migration needs to INSERT into a table, it must first ensure the table exists with `CREATE TABLE IF NOT EXISTS`. Use idempotent inserts (`ON CONFLICT DO NOTHING` for PostgreSQL, `INSERT OR IGNORE` for SQLite).
- **Case-sensitive DB comparisons**: PostgreSQL string comparison is case-sensitive. Always use the canonical (uppercase) user ID from the database (e.g., `user.User_id.String()`) for permission and lookup queries — never use raw user input directly.
- **Composite primary keys**: Tables can have composite primary keys (e.g., `User_Member` has `user_id + role_id + company`). `TableMetadata` stores `map[string][]string` for all PK fields. The API uses comma-separated PK values in URLs (e.g., `/delete/HANS2,READER,TEST-COMPANY`), parsed by `parseRecordKey()` in the backend.

## Frontend Conventions

- **TypeScript strict mode** with `checkJs` enabled
- **Suppressed Svelte warnings**: `a11y_no_noninteractive_tabindex`, `a11y_no_noninteractive_element_interactions` (intentional tabindex patterns), `css_unused_selector` (Tailwind `@apply`)
- **Tailwind with BC/NAV color scheme**: `text-nav-blue` (#002050), `text-nav-lightblue` (#4472c4), font: Segoe UI. Dark mode via `class` strategy.
- **Svelte 5 runes**: Always use runes (`$state`, `$derived`, `$effect`, `$props`, `$bindable`) — never legacy Svelte 4 patterns (`export let`, `$:`, `$store` syntax).
- **i18n**: All labels and field captions in the frontend must come from backend translation files — never hardcode display text in Svelte components.
- **BC/NAV keyboard shortcuts**: Ctrl+N new, Ctrl+E edit, Ctrl+D delete, Ctrl+S save, Ctrl+F find, F5 refresh, Escape cancel, Ctrl+Home/End first/last, PageUp/Down prev/next
- **Code field behavior (ABSOLUTE RULE)**: Fields with `types.Code` must allow typing in any case, then auto-uppercase the value on blur (when the field loses focus). This matches standard NAV/BC behavior. Never force uppercase while typing — only convert on exit. Apply this everywhere Code fields are rendered: login forms, list page edit cells, card page fields, modal dialogs, and any other input bound to a Code field.

## Generic List Page Behaviors (ABSOLUTE RULES)

All list page behaviors are driven by page metadata — never add table-specific logic in `ListPage.svelte` or `PageRenderer.svelte`.

- **New record initialization**: When clicking New, all repeater fields are initialized to `""` (empty string). This ensures composite PK fields are defined from creation. Only one empty new row can exist at a time — clicking New again focuses the existing empty row.
- **Delayed insert (NAV `DelayedInsert`)**: New records are inserted only when the user **leaves the row** — not when leaving an individual cell. This allows the user to fill all fields (including optional PK fields like `company`) before the insert fires. The insert triggers when focus moves to a different row or outside the table. Required PK fields must have non-empty values; optional PK fields (without `required: true`) can remain blank. The `required` flag is sent from the backend via table YAML metadata → `TableMetadata` → page field definitions.
- **Lookup fields in edit mode (ABSOLUTE RULE)**: Fields with `table_relation` that have `columns` + `rows` (advanced lookup) must render using the `LookupDropdown` component with `compact={true}` — providing a table-style dropdown with column headers, type-ahead filtering, and keyboard navigation. Fields with only `simple` lookup data render as `<select>`. Fields with no lookup render as plain `<input>`. Never use `<datalist>` for lookup fields. The `LookupDropdown` must be wrapped in a `<div data-row data-col>` container for focus management.
- **LookupDropdown select vs blur (ABSOLUTE RULE)**: When the user selects a value from a `LookupDropdown`, the component must call `onselect` (to set the value) and re-focus its input — it must NOT call `onblur` or trigger a save. The save (`handleCellBlur`) fires only when focus actually **leaves** the component (e.g., user presses Tab). This prevents premature inserts during data entry and is critical for delayed insert behavior on composite PK tables.
- **Composite key encoding**: `getRecordId()` joins all PK values with commas for composite keys. Empty string is a valid PK value (e.g., blank company = all companies access). The function accepts `primaryKeyFields` array for composite support.
- **Cell blur auto-save**: On blur, existing records call `modifyRecord`. New records are only saved when the user has left the row (see delayed insert above). The `isSaving` guard must be set **before** any async validation to prevent race conditions from concurrent blur events. Fields using `LookupDropdown` (advanced lookup) skip server-side validation in `handleCellBlur` since the component validates internally. Failed saves revert to original values.
- **Focus management**: `focusCell()` must handle three cell types: direct `<input>` elements, `<select>` elements, and `LookupDropdown` wrapper `<div>` containers. It tries `input[data-row][data-col]` and `select[data-row][data-col]` first, then falls back to `div[data-row][data-col]` and focuses the `<input>` inside it.
- **Keyboard navigation**: Arrow keys move between cells, Tab/Shift+Tab move horizontally, Enter on the last row creates a new row. Escape exits edit mode.

## API Response Format

```
Success: { "success": true, "data": { ... }, "captions": { "table": "...", "fields": { ... } } }
Error:   { "success": false, "error": "..." }
List:    { "success": true, "data": { "records": [...], "total": N, "page": N, "page_size": N }, "captions": { ... } }
```

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
