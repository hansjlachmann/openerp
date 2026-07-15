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
- [ ] Request validation (field constraints)
- [ ] HTTPS/TLS support
- [ ] API versioning
- [ ] WebSocket support for live updates
- [ ] Pagination for large datasets
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

_Not tracked here (identified as noise, not real TODOs): i18n `%1`/`%2` substitution logic,
SQL `$N`/`?` parameter builders, Svelte input `placeholder=` attributes, UUID templates in
`toast.ts`, and `vi.stubGlobal` test helpers._
