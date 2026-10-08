# TODO

Consolidated backlog for OpenERP. This is the single source of truth for outstanding
work, gathered from sub-READMEs, `docs/migrations.md`, and inline code markers.
`file:line` references are clickable — verify them before starting, as line numbers drift.

Legend: `- [ ]` open · `- [x]` done. Group headings map to areas of the codebase.

---

## ⭐ First priority (next session, from 2026-10-07)

- [x] **Demo (k3s, repo `pi-cluster`): `jwt-secret` added to `openerp-secret` with sops** (2026-10-08,
      pi-cluster `3026f62`). Backend pod has `JWT_SECRET` (64 chars), no "No JWT_SECRET" warning; logged in,
      deleted the backend pod, reload → still logged in. Notes for next time: the age key is not on the
      workstation — load it per shell with
      `export SOPS_AGE_KEY=$(kubectl -n flux-system get secret sops-age -o jsonpath='{.data.age\.agekey}' | base64 -d)`;
      Flux reverts `kubectl rollout restart` — restart with `kubectl -n openerp delete pod -l app=openerp-backend`.
- [x] **Production: upgraded to 0.1.84 and switched to `prod.env` / `scripts/prod.sh`** (2026-10-08).
      Backup first; migration 7 applied (index count in the dump 408 → 223, all tables' data present);
      new `JWT_SECRET`, sessions survive a backend restart; `PUBLISH_HOST=127.0.0.1` (users come in
      through nginx on 443); `restart: unless-stopped` now active. Daily backup cron (02:15,
      `scripts/backup.sh`) installed and tested.
- [ ] **Production: optionally rename the company with the invalid technical name** (keep its Display Name).
- [ ] **Production: copy the backups to another machine** — `~/backups` is on the server's own disk.
- [ ] **Production: upgrade to 0.1.85** once its release is out — needs `git pull` (the compose file passes
      `NAV_PROXY_URL`). `NAV_PROXY_URL` is already set in the server's `prod.env` (2026-10-08); the NAV
      proxy address is no longer built in (`docs/operations.md`, "NAV report proxy"). First merge the
      release PR "chore(main): release 0.1.85" and wait for the Build & Release run to push the images,
      then on the server:
      ```bash
      cd ~/openerp && scripts/backup.sh && git pull
      sed -i 's/^APP_VERSION=.*/APP_VERSION=0.1.85/' prod.env
      scripts/prod.sh pull && scripts/prod.sh up -d
      ```
      Check: `scripts/prod.sh ps` (backend/frontend on 0.1.85), `scripts/prod.sh logs backend --since 5m |
      grep -iE "migration|error|WARNING"`, the browser shows v0.1.85, and a NAV report Job Queue entry runs.
- [ ] **Production: change `POSTGRES_PASSWORD`** from the default `openerp` (`docs/operations.md`,
      "Changing the database password"); lower risk now that 5432 is localhost only.

---

## Feature: SIFT (Sum Index Fields) — Phase 1: foundation ✅ DONE

Implemented as planned below (`backend/foundation/sift`, tablegen, `SyncKeys`); first key: Customer
Ledger Entry `customer_open`. Verified:
- Tests (SQLite + Postgres): 400 random inserts/modifies/key moves/deletes/rollbacks, NULL keys, bulk
  statements, TRUNCATE → totals always equal the entries; 8 parallel writers on one totals row
  (some rolled back) → exact; rebuild on added/removed sum field, removed key, missing totals table;
  `DropCompany`; FlowFields via SIFT equal sums over the entries (also after Modify/Delete).
- demo04 (Postgres, 141,710 entries): totals built at startup in 0.75 s (17,272 totals rows); all
  10,000 customers' Balance/Sales/No. of entries identical to before; a FlowField reads 1 totals
  row via its primary key (0.1 ms); insert/close/delete of an entry through the API moves the
  totals correctly; second start rebuilds nothing; removing `sales_lcy` from `sum_index_fields`
  → warning + rebuild + fallback to the entries, adding it back → rebuild, totals match.

Next phases: 2 heavy demo data and measurement ✅ done → 3 Verify/Rebuild SIFT codeunit ✅ done →
4 FlowFilter fields (Date Filter) ✅ done.

**Phase 3** — codeunit 50110 Verify SIFT (Job Queue, parameter VERIFY / REPAIR), `sift.VerifyKey` /
`RebuildKey`, generated `VerifySIFT`. Tested on SQLite/Postgres and through the Job Queue on demo05
(corrupted totals detected and logged as Error, REPAIR rebuilds, verify passes again).

**Phase 2** — demo data size HEAVY: 200 customers, 190,062 entries (~1,000 per customer, two years).
Measured on demo05 (Postgres), customer C00010 with 1,012 entries, after VACUUM:
| Query | Sum over entries | SIFT totals |
|---|---|---|
| Balance (open entries) | 1.2–1.9 ms | 0.35–0.42 ms |
| Sales, all time | 5.8–11.3 ms | 0.23–0.27 ms (~25–45×) |
| Sales 2025 (date filter) | 1.8–6.1 ms | 2.3–2.6 ms |
| Balance Q2 2026 (date filter) | 0.4–1.0 ms | 0.4–0.5 ms |
Findings:
- The date key was first (customer, open, date): a date-filtered Sales read all day rows of the customer
  (slower than the entries). Reordered to `customer_date_open` (customer, date, open) — the change was
  rebuilt automatically for all companies (rebuild path verified for real).
- Date-filtered SIFT gives no gain here: a customer gets ~1–2 entries per day, so the date key has 584 day
  rows for 1,012 entries. It pays off only with many entries per day (G/L volumes). Decided: `customer_date_open`
  removed (migration 006 drops its index); date-filtered Customer FlowFields sum the entries.
- Right after the bulk load the totals had ~70 dead row versions per row (every posting updates the totals
  row); after autovacuum/VACUUM the reads are fast.
- [x] Loading HEAVY took 9 min (row-by-row Insert, two SIFT triggers per entry). Now the ledger entries
      go through the generated `InsertAll` (multi-row INSERT per 1,000 entries, OnInsert still runs) with
      their SIFT totals suspended and built once at the end (`sift.Suspend` + `SyncKeys`, same
      transaction). Local Postgres: 5 min 21 s → ~1 min (the rest is Postgres inserting 190,000 rows
      into five indexes).
- [x] **Performance: dead row versions in SIFT totals tables (Postgres).** Every posting UPDATEs its totals
      row; MVCC left the old version until autovacuum ran (demo05 after HEAVY: ~70 row versions per totals
      row, a totals read no faster than summing the entries). Fixed:
      1. Totals tables are created `WITH (fillfactor = 50)` and per-table autovacuum thresholds
         (`autovacuum_vacuum/analyze_scale_factor = 0`, `…_threshold = 1000`); `Sync` sets them on
         existing totals tables (`ALTER TABLE … SET`, verified on all local companies at startup).
         Updates are HOT: 78 % of 7,602 totals updates in one bulk UPDATE.
      2. Bulk loads build the totals once (`sift.Suspend` + `SyncKeys`): after loading HEAVY the totals
         table has 400 live and 0 dead rows (before: 189,662 dead).
      3. `RebuildKey` (Verify SIFT REPAIR) runs `VACUUM ANALYZE` afterwards; a build runs `ANALYZE`.
G/L Account / G/L Entry are out of the current scope.

## SIFT Phase 4: FlowFilter fields (Date Filter) ✅ DONE

Implemented as planned below. Verified: parser tests per type; on demo data the Customer FlowFields
with a Date Filter (quarter, open ranges, two months with |) equal sums over the entries in the period,
card and list; API 400 on bad expressions/unknown fields; date conversion tests (nb-NO, da-DK, en-US, t,
|, <>, invalid dates). Browser on demo04: Date Filter 04/01/26..06/30/26 → C00010 balance 0, sales
48,992.54 (same as the database), modal card shows the period value, invalid date → message, clear →
back. Second SIFT key built at startup (demo04: 133,309 totals rows in 2.2 s — the demo spreads ~14
entries per customer over a year, so few entries share a day). Found on the way: a broken
messages.yaml made all messages show as keys (loader only warns) — new test parses every translation
file; Enter in the filter pane also opened the selected record — fixed with `data-own-keys`.

### Goal
NAV/BC FlowFilter fields (FieldClass FlowFilter): not stored, they hold a filter the user sets, and
FlowFields apply it in their CalcFormula (`Posting Date = FIELD(Date Filter)`). Types: Date, Boolean,
Integer, Text, Code. First use: Customer "Date Filter" applied to Posting Date in Balance (LCY), Sales
(LCY) and No. of Ledger Entries.

### Decisions (with the user)
- The user sets FlowFilters in the list's filter pane, section "Filter totals by" (BC). A modal card
  opened from the list uses the same FlowFilters.
- BC syntax in the user's date format: `01.01.26..31.03.26`, open ranges (`..31.03.26`, `01.01.26..`),
  single date, `|` alternatives, `t` = today. The frontend converts dates to ISO; the backend only sees ISO.
- Second SIFT key `customer_open_date` (customer_no, open, posting_date), same sum fields: no date
  filter → `customer_open` (1–2 rows per customer); date filter → the date key.

### Design
- YAML: `flow_filter: true` on a field (type types.Date / bool / int / types.Text / types.Code); a FlowField
  uses it with a flow filter of type `filter`: `{field: posting_date, type: filter, value: date_filter}` —
  applied when the FlowFilter has a value, ignored when empty.
- Generated code: FlowFilter fields are not columns (no schema/ToMap/FromMap/search/sort); the record keeps
  each FlowFilter's expression (string). `SetFlowFilter(field, expr) error` (validates by parsing) and
  `GetFlowFilterFields()` on the Table interface.
- `backend/foundation/flowfilter`: one parser `Clause(column, kind, expr) (sql, args, error)` — `|`
  alternatives; `a..b`, `..b`, `a..`; `<>x`; `*` wildcards for Text/Code (Code uppercased); Yes/No/true/false
  for Boolean; ISO dates; numbers for Integer. Typed bind parameters only.
- FlowField calculation (card `CalcFields`, list `CalcFieldsForRecords`, Sum and Count): `filter` flow filters
  add the parsed clause; tablegen resolves two SIFT keys per FlowField (without and with the filter fields)
  and the generated code picks at run time.
- API: `flow_filters` (JSON `[{field, expression}]`) on `/list` and `/card`; unknown FlowFilter or parse error →
  400 with the message. Page API sends `flow_filter_fields` (name + type).
- Frontend: FilterPane "Filter totals by" section (FlowFilter fields, captions from translations); PageRenderer
  keeps `currentFlowFilters` and sends them; local-date → ISO conversion in a util; filter badge counts them;
  ListPage passes them to the modal card.

### Verification
- Parser unit tests per type (ranges, open ranges, `|`, `<>`, wildcards, errors).
- SQLite demo data: Balance/Sales/No. of entries with a date filter equal direct sums over entries in that period
  (card and list paths); without filter unchanged; SIFT date key used when filtered.
- API: `flow_filters` on list/card, 400 on bad expressions/unknown fields. Frontend unit test for date conversion.
- Browser on demo04: set Date Filter in the filter pane → totals change; clear → back; modal card shows period values.

### Original plan

### Goal
FlowFields over large ledgers (100,000+ entries per customer or account) must not add up every entry
on each read. As in NAV/BC (SumIndexFields; BC: indexed views), table keys can declare sum fields;
the database keeps a totals table per key up to date, and FlowFields read the totals.

Phases: **1 foundation (this plan)** → 2 LARGE+ demo data (~1,000 entries per customer) and
measurement → 3 Verify/Rebuild SIFT codeunit (Job Queue) → 4 FlowFilter fields (Date Filter).

### Current state (reuse, don't rebuild)
- FlowFields: `calc_formula` Sum/Count + `flow_filters` (const / field) in the table YAML; generated
  per record `calcSum…`/`calcCount…` and batched `calcForRecords_…` (`tools/tablegen/main.go`).
  Both run `SUM(...)`/`COUNT(*)` over the source table today.
- Keys: `keys:` in the YAML → `CREATE INDEX` in the generated `CreateTableWithDBType` — **only when
  the table is created**; table sync (`backend/foundation/objects/initializer.go`
  `InitializeCompanyTablesWithDBType`) adds missing columns to existing tables but no indexes.
- Decimals: Postgres `NUMERIC` (exact), SQLite `TEXT` (summed as float). Booleans: Postgres
  `BOOLEAN`, SQLite `INTEGER`.

### 1. YAML: sum index fields on keys
```yaml
keys:
  - name: customer_open
    fields: [customer_no, open]
    sum_index_fields: [remaining_amt_lcy, sales_lcy, amount_lcy]
```
- tablegen `Key` gets `SumIndexFields []string`. Validation at generation time (fail the build):
  key and sum fields must exist and be stored (not FlowFields); sum fields must be Decimal or int.
- Apply to `custledgerentry.yaml` key `customer_open` (sums: `remaining_amt_lcy`, `sales_lcy`,
  `amount_lcy`). One key serves both customer FlowFields (Balance filters customer + open,
  Sales filters customer only → sums the open and closed totals rows).

### 2. Totals table per SIFT key (generated)
- Name `company$Table$SIFT$key` (global tables: `Table$SIFT$key`). Postgres limits names to 63 bytes:
  when longer, use `<first 50 bytes>_<8-char hash>`; same rule for trigger/function names. One
  generated helper `siftObjectName(company, key)` — never build these names elsewhere.
- Columns: the key fields (same types as the source; together the PRIMARY KEY), one column per sum
  field (`NUMERIC` / SQLite `REAL`; int → `BIGINT`/`INTEGER`), and `cnt BIGINT NOT NULL`.
- One row per distinct key value combination; rows with `cnt = 0` are deleted.

### 3. Maintenance by database triggers (same statement / transaction as the entry)
Generated per SIFT key, for both databases:
- **Postgres:** one PL/pgSQL function + `AFTER INSERT OR UPDATE OR DELETE … FOR EACH ROW` trigger:
  - INSERT → `INSERT INTO sift (keys, sums, cnt) VALUES (NEW.…, NEW.…, 1) ON CONFLICT (keys) DO UPDATE
    SET sum = sift.sum + EXCLUDED.sum, cnt = sift.cnt + 1`
  - DELETE → `UPDATE sift SET sum = sum - OLD.…, cnt = cnt - 1 WHERE keys = OLD.…`, then
    `DELETE FROM sift WHERE keys = OLD.… AND cnt = 0`
  - UPDATE → only when a key or sum column changed (`AFTER UPDATE OF <cols>` +
    `WHEN (OLD.(…) IS DISTINCT FROM NEW.(…))`): the DELETE step for OLD, then the INSERT step for NEW.
  - Statement-level `AFTER TRUNCATE` trigger → `TRUNCATE` the totals table.
  - Only deltas (`sum = sum + x`), never read-and-write-back: concurrent postings to the same key
    serialize on the totals row lock and no update is lost; a rollback undoes the totals with the entry.
- **SQLite:** three triggers (`AFTER INSERT` / `AFTER DELETE` / `AFTER UPDATE OF … WHEN …`) with the same
  statements (upsert `ON CONFLICT … DO UPDATE` is supported by the bundled SQLite of go-sqlite3
  v1.14.24). SQLite has no TRUNCATE.
- NULL-safe: sums use `COALESCE(x, 0)`; key columns compared with `IS NOT DISTINCT FROM` (Postgres) /
  `IS` (SQLite) where a key value can be NULL.

### 4. Create, sync and initial fill
- New generated method `EnsureSIFT(db, company, dbType) error` on every table (no-op without SIFT keys);
  `objects/initializer.go` calls it (optional interface) for **existing and new** tables, after
  column sync. Also fixes the missing-index gap: `EnsureSIFT` runs `CREATE INDEX IF NOT EXISTS` for all
  keys, so keys added later reach existing tables.
- Definition fingerprint (key fields + sum fields + generator version) stored in a global table
  `_sift_definition (company, table_name, key_name, fingerprint)`. Missing or different →
  rebuild in **one transaction**: drop triggers/function/totals table, create the totals table,
  create triggers, fill with `INSERT INTO sift SELECT keys, SUM(…), COUNT(*) FROM table GROUP BY keys`,
  store the fingerprint. Creating the trigger locks the source table against writes until commit
  (Postgres `SHARE ROW EXCLUSIVE`), so no entry can slip in between fill and trigger.
- Unchanged fingerprint → nothing to do (startup stays fast). Removed SIFT key → drop its objects.
- Changing `sum_index_fields` later is a normal change: **adding** a field (e.g. a new amount column)
  → column sync adds the column first (EnsureSIFT runs after it), the fingerprint differs → rebuild
  with the new sum column, filled from the entries. **Removing** a field → rebuild without it (the
  entry column and its data stay). Same for changed key fields or a renamed field.
- Runs per company in the existing startup sync; check how multi-pod startup is serialized (migrations
  have a distributed lock; table sync does not yet) — reuse the migration lock if needed.

### 5. FlowFields read the totals
- tablegen picks, per FlowField, a SIFT key on its `source_table` whose fields contain every flow
  filter field and (for Sum) whose `sum_index_fields` contain `source_field`. Prefer the key with the
  fewest fields. None found → today's code (sum over the entries), unchanged, and tablegen prints a
  warning ("FlowField X: no SIFT key covers it, sums the entries") so a removed sum field is noticed.
- Generated code with a SIFT key:
  - card `calcSum…`: `SELECT COALESCE(SUM(sum_col), 0) FROM sift WHERE <flow filters>`
  - `calcCount…`: `SELECT COALESCE(SUM(cnt), 0) …`
  - list `calcForRecords_…`: the same grouped query as today, against the totals table (`GROUP BY` the
    field filter column, `IN` list ≤ 500 keys, whole table above).
- tablegen needs the source table's definition for this (FlowField on Customer → keys of Customer
  Ledger Entry): load all YAML definitions first, then generate (it currently processes files one
  by one).

### 6. Out of scope for phase 1
FlowFilters / date filters (phase 4), Min/Max/Average (not summable incrementally), `CalcSums` API for
codeunits (later), SIFT on global tables beyond naming support, MaintainSIFTIndex = false variants.

### Files to create / modify
- `tools/tablegen/main.go` — `Key.SumIndexFields`, validation, two-pass YAML loading, SIFT DDL +
  trigger templates (Postgres/SQLite), `EnsureSIFT`, `siftObjectName`, FlowField SIFT read path.
- `backend/foundation/objects/initializer.go` — call `EnsureSIFT` for existing and new tables.
- `backend/foundation/tables/interface.go` (or an optional `SIFTMaintainer` interface).
- `backend/business-logic/tables/definitions/custledgerentry.yaml` — `sum_index_fields` on `customer_open`.
- regenerated `backend/generated/tables/*`; CLAUDE.md (FlowFields / SIFT rules, naming, "only deltas").

### Verification
- **Unit (SQLite, `go test -race`):** random sequence of inserts, modifies (amount, open, customer_no
  changes), deletes, `DeleteAll`/`ModifyAll`, transactions rolled back → totals table equals
  `GROUP BY` over the entries (tolerance 0.005 on SQLite floats); `cnt = 0` rows removed;
  FlowField values via SIFT equal the old per-entry sums (card and list paths).
- **Postgres (optional test, runs when `TEST_POSTGRES_DSN` is set; run locally against the docker db):**
  same consistency test, exact; concurrency test — N goroutines posting to the same customer in
  parallel transactions, some rolling back → totals exact; TRUNCATE empties the totals.
- **Sync:** fresh company → SIFT created and filled; existing company (demo04, 141,710 entries) →
  built once at startup (measure time), second start does nothing; changed `sum_index_fields` →
  rebuilt; a sum field added to the table and to `sum_index_fields` → rebuilt with the new column;
  a sum field removed → rebuilt without it, the FlowField falls back and the warning is printed;
  name > 63 bytes → hashed name works.
- **End to end on demo04:** Customer list and card show the same Balance/Sales as before; EXPLAIN shows
  the totals table used; demo data loader (one big transaction) still works and leaves exact totals.

---

## Performance with large data ✅ DONE (steps A and B; open follow-ups under step B)

Found with the LARGE demo data set (10,000 customers, 141,710 customer ledger entries, local
Postgres, company `demo04`). Measured on the list API exactly as the frontend calls it:

| Request | Time | Size |
|---|---|---|
| Customer list as the page loads it (all rows, with FlowFields) | **28.4 s** | 3.8 MB |
| Customer list, same columns without FlowFields | 0.4 s | 3.6 MB |
| Customer list, one page of 100 rows with FlowFields | 0.3 s | — |
| Customer Ledger Entries list as the page loads it (all rows) | **14.4 s** | **132 MB** |
| One `GROUP BY customer_no` query for the balance of all customers | < 0.4 s | — |

### Cause 1 — FlowFields are calculated per row (N+1 queries)
`ListRecords` (`backend/api/handlers/tables.go`) calls `table.CalcFields(flowFields...)` for every
row, and each FlowField is its own query (`calcSum…`/`calcCount…` in the generated `*_base.go`).
10,000 customers × 3 FlowFields = 30,000 queries ≈ 28 s. It also calculates every FlowField, not
just the requested ones (`no_of_ledger_entries` is not on the list).

### Cause 2 — list pages always load the whole table
`PageRenderer.loadListData()` never sends `page_size`, so the backend returns every row (the
paging support in `ListRecords` + generated `SetPage` is unused). Search, sort and keyboard
navigation in `ListPage.svelte` work client-side on the full array, and every row is rendered.
141,710 ledger entries = 132 MB JSON and a DOM the browser cannot handle.

### Step A — batch FlowField calculation ✅ DONE
- [x] tablegen generates `CalcFieldsForRecords(records, fields...)` (in `tables.Table`): per FlowField
      with one `field` flow filter, one `SELECT key, SUM/COUNT … GROUP BY key` — with an `IN` list for
      up to 500 keys, over the whole source table for more. Other FlowFields fall back to per record.
- [x] `ListRecords` reads all rows first, then batch-calculates only the requested FlowFields
      (`flowFieldsToCalc`; all when no `fields` param). The card keeps `CalcFields`.
- [x] Tests: values identical to per-record `CalcFields` on demo data (IN-list and whole-table
      paths, customers without entries), only requested fields calculated.
- [x] Measured on demo04 (Postgres): Customer list 28.4 s → **1.1–1.4 s**; a page of 100 rows
      26 ms; sum of all balances equals the ledger total.

### Finding after step A — the browser needs ~40 s to render 10,000 rows
Measured in Chromium (Playwright, production build) on demo04: the Customer list data arrives after
1.4 s, but the rows appear only after **41 s**. Profile: garbage collection and Svelte's per-cell
`{#if}` blocks (each cell renders the full state/type branch chain) ~20 s, the first layout of the
10,000-row table (forced in `scrollRowIntoView`) ~6 s, DOM inserts ~4 s. Rendering every row cannot be
made fast enough — the list must render only the visible rows. Tried and rejected: `untrack` on the
`{#each}` key (helps the dev build only, no change in production).

### Step B — windowed list loading ✅ DONE
Lists load only the rows that fit on the page plus two pages above and below (max 200), with
`offset`/`limit`; search and sort run on the server over all records (see CLAUDE.md "Windowed
loading"). Measured on demo04 in Chromium: Customer list rendered in **0.94 s** (was 41 s).
Verified in the browser: End/Ctrl+End/Home/Ctrl+Home across 10,000 records, PageDown and held
ArrowDown across windows, mouse-wheel loading, server search and sort, editing a cell across a
window edge (saved), ArrowDown on the last record opens a new row.

Open follow-ups:
- [x] **Focus lost after editing a list cell** — fixed: the blur of the removed edit input was taken
      as "left the table" (double save + navigation mode, whose delayed page focus stole the focus).
      Fixed together with: rows keyed by the live primary key (typing a new No. saved a half-typed
      rename after one character), unchanged rows MODIFYed on every cell move (2 requests per key,
      hit the 300/min rate limit), a single-slot save queue that dropped edits, characters lost by
      the 50 ms focus delay, and clicking the search box while editing not saving the cell.
- [x] Cell modes losing arrow keys during a window load — fixed by the above (180 ArrowDowns now
      land on row 181).
- [x] Renaming a record updates related records (BC/NAV Rename): generated `renameReferences` from the
      YAML table relations, in one transaction with the modify. Verified: C00030 → C00030X moves all 36
      entries (and SIFT totals), a rename to an existing No. rolls back completely.
- [x] Renaming a **Company** record moves its data: `Company.OnRename` renames all `old$…` tables,
      indexes and sequences, rebuilds SIFT totals, updates User Member — one transaction, verified on
      SQLite and Postgres incl. rollback after a late failure. On the way: `OnRename` is now called by
      every generated `Modify` when a key changes; company delete via the API drops the SIFT objects
      too and no longer uses `LIKE 'name$%'` (deleting `a_b` dropped the tables of `axb`); sessions of a
      renamed/deleted company are dropped (the renaming user keeps working with a new cookie); startup
      no longer re-creates a renamed `COMPANY_NAME`; global tables no longer get an index copy per
      company (migration 007: 9 → 2 indexes per global table locally).
- [ ] After renaming the company you work in, the menu bar shows the old company name until the next
      page load (the server side follows at once). Needs a generic way for the list page to refresh
      the session after a save.
- [x] A rename to an existing key failed with the generic "Failed to modify …": `ModifyRecord` now checks
      the new full primary key first and answers 409 "<Table> <key> already exists".
- [x] Code fields in list cells were sent to the API lowercase (`fieldTypes` holds `types.Code`, the
      uppercase check compared with `code`) — fixed with `isCodeType` (`utils/fieldHelpers.ts`).
- [x] Customer Ledger Entries page 25 (menu, drilldowns from the Customer list's Balance/Sales, card
      action "Ledger Entries"/Ctrl+F7 filtered to the customer). Found on the way: actions without
      `enabled: true` were disabled (Go bool default) and pages naming their table by registry name got
      no primary key fields — both fixed.
- [ ] Page 25 is editable (to try the on-demand customer lookup); make it read-only once posting
      exists (BC: posted ledger entries are not editable).
- [x] New User (modal card) failed with "User number cannot be empty" on Windows (Chrome and Edge):
      the browser's password manager filled the saved login into the card's password field before a
      User ID was typed, and the card inserted at once, then blocked the modal. Cards (modal and page)
      now insert a new record only when every primary key field has a value (`hasPrimaryKey`); record
      inputs set `autocomplete` (`new-password` for masked fields, `off` otherwise). Fixed in 0.1.82,
      confirmed on the demo environment.
- [x] Deleting a record on an editable list (e.g. Users) after clicking a row left it on screen until
      a reload: in cell-selected mode the list shows its editable copy (`editableRecords`), which was not
      rebuilt from the reloaded window. `handleDelete` now rebuilds it (keeping unsaved new rows) and
      keeps the selection on the next record. Delete in the modal card used the browser's English
      `confirm()` and left the record count stale: now the translated dialog and a list reload.
- [ ] List cells ignore per-field `editable: false` except for paste and F8 (typing/F2 still edit).
- [x] Lookups on demand for large related tables (> 200 rows): `lazy_url` + `GET …/lookup/:field`
      with server search; relation values checked on insert/modify. demo04 Customer Ledger Entries
      window 422 ms → 55 ms (deep window 865 ms → 88 ms).
- [ ] Card navigation (`/ids`) returns all keys in primary-key order and ignores the list's
      sort/search/filter.
- [ ] SQLite `LOWER()` only folds ASCII, so on SQLite the search is case-sensitive for æ/ø/å
      (Postgres is fine).

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
- [x] **Security: `password_hash` was returned by the API.** Fixed with a YAML field flag
      `sensitive: true` (tablegen → `FieldInfo.Sensitive`): the table API strips sensitive fields from
      every record it returns (`ftables.PublicMap`), rejects them in filters, sort and search
      (`IsQueryableColumn` — a `LIKE` search on the hash would reveal it piece by piece) and in
      insert/modify/validate payloads (the hash could be overwritten with a known one); tablegen rejects
      relations whose dropdown shows one. The password is still set through the virtual `password` field.
- [x] SMTP password (`SMTP_Setup.password`) was returned in plain text. Now **masked** (YAML
      `masked: true`, BC ExtendedDatatype Masked): write-only — the API sends `••••••••` when set, the
      placeholder sent back keeps the stored value, "" clears it, no filter/sort/search on it. Every
      masked field (table YAML, or page YAML for the User card's virtual password) renders as a
      password input on cards, list cells and modal cards (no more `field.source === 'password'`).
- [ ] The SMTP password is still **stored** in plain text in the database (anyone with database
      access or a backup can read it). Encrypt masked fields at rest with a server key (not
      `JWT_SECRET`; e.g. `SECRETS_KEY`, AES-GCM), decrypted only by the mailer.
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
- [x] Codeunits broke on PostgreSQL: they opened tables with `.Init(db, company)`, which always used the
      SQLite database type (`?` placeholders). The three affected codeunits (50000–50002) were old CLI
      leftovers that nothing registered or called — deleted. `Init` itself is removed from the `Table`
      interface, the tablegen templates and the wrappers, so only `InitWithDBType` exists and the
      compiler rejects the mistake. (The table relation validators had the same bug; fixed earlier.)

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

- [ ] **Copy/paste records on list pages (low priority).** Only on list pages that allow it: new page
      YAML property `copy_paste_allowed: true` (CopyPasteAllowed). The user marks the records to copy
      (multi-row selection), then pastes them; every pasted record is inserted (BC/NAV Insert, triggers
      run). Two scenarios:
      1. **Same table, new keys (composite primary key):** mark records, set a filter on the primary key
         field(s) that should differ (e.g. a new role or company in User Member), paste — the filter's
         values replace those key fields in the pasted records. A record whose resulting key already
         exists is an error (report which).
      2. **Between companies, same table:** mark records, switch company (Ctrl+O), open the same list
         and paste. The records keep their primary keys, which must not exist in the target company yet
         (error otherwise).
      Open points: what to do when some records of a paste fail (all-or-nothing transaction is the
      safer default), where the copied records are held across the company switch (server side vs.
      browser storage), sensitive fields never copied, FlowFields not copied.

- [ ] `frontend/src/lib/components/pages/PageRenderer.svelte:500` — only `List` and `Card`
      page types render; all other types hit the "not yet supported" fallback. Add support
      for additional page types as they are introduced.
- [ ] WebSocket support for real-time updates (pairs with the backend WebSocket item).
- [ ] Keyboard Shortcuts help window (`frontend/src/routes/help/shortcuts/+page.svelte`):
      remove the intro text "Keys work as in Microsoft Dynamics NAV and Business Central."
      (the `HELP_INTRO` line under the title, and the `HELP_INTRO` key in
      `translations/{en-US,nb-NO,da-DK}/messages.yaml` and the `HELP` constant).
- [x] Create Go API backend — done
- [x] YAML page definitions (dynamic page generation) — done
- [x] Generic `PageRenderer` (List/Card) — done

## Migrations

From `docs/migrations.md`.

- [x] SQLite-safe drop-column / alter-column migration helper — done via `RecreateTable`
      + SQLite `ChangeColumnType` in `backend/foundation/migrations/helpers.go`.

## Deployment / Operations

Gaps in the production setup (`docker-compose.prod.yml`), found while recovering a server after
an OS upgrade and reboot.

- [ ] **CI: the frontend Docker image build sometimes hangs.** The "Build and push frontend" step
      (multi-arch, arm64 emulated with QEMU) hung in the 0.1.68 release (cancelled after GitHub's 6 h
      job limit) and in 0.1.82 (cancelled after ~45 min; re-running only the Docker job finished in
      4 min). Add `timeout-minutes: 30` to the Docker job so a hang fails fast and can be re-run; if it
      keeps happening, build arm64 on a native runner instead of under QEMU.
- [x] **Auto-start after reboot** — all services in `docker-compose.prod.yml` have
      `restart: unless-stopped`.
- [x] **Only expose nginx** — db on `127.0.0.1:5432` (SSH tunnel for admin tools); backend/frontend
      on `PUBLISH_HOST` (default `127.0.0.1`, for a tunnel/proxy on the same host; `0.0.0.0` only if a
      proxy on another machine needs them); nginx 80/443 as before.
- [x] **Postgres credentials** — read from `POSTGRES_PASSWORD` in the untracked `prod.env` (default
      `openerp`, so existing installations keep working); changing it on an existing volume
      (`ALTER USER` first) is in `docs/operations.md`.
- [x] **`JWT_SECRET` in production** — set in `prod.env` (`openssl rand -hex 32`); the backend still
      only warns when it is missing (user decision: no refusal to start).
- [x] **Automated backups** — `scripts/backup.sh` (`pg_dump -Fc` via the db container into
      `~/backups/openerp-auto-*.dump`, prunes only those after 14 days) + cron example. Verified
      locally: dump restored into a scratch database matches the live one. Copying backups off the
      host stays a separate, site-specific step.
- [x] **Operations doc** — `docs/operations.md`; production runs through `scripts/prod.sh`
      (`docker compose --env-file prod.env -f docker-compose.prod.yml`), so nothing tracked is edited on
      the server and `git pull` no longer conflicts.
- [x] Switch production to `prod.env` and set up the backup cron (2026-10-08; the demo runs on k3s, not
      compose). The off-host copy is open under "First priority".
- [ ] **(Medium priority) Remove installation-specific details from the git history.** Older commits
      (code and two commit messages) contain details of a customer installation; the code no longer
      does (`NAV_PROXY_URL`). Rewrite with `git filter-repo` (replace-text + replace-message), force-push
      `main`, the affected tags and branches; then reset every clone (`git fetch && git reset --hard
      origin/main`), ask GitHub Support to purge cached commits and PR refs, and fix the commit links in
      `CHANGELOG.md`. The released backend images up to 0.1.84 contain the same detail: once production
      runs a newer release, delete those image versions (or make the package private).
- [ ] **(Low priority) Production: renew the TLS certificate before it expires** — replace the
      certificate and key in `certs/` on the server, then `scripts/prod.sh restart nginx`.

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
encrypt the SMTP password at rest (masked in the API since 0.1.81, still stored in plain text).

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

## Feature: Demo Data — step 1 ✅ IMPLEMENTED

Codeunit 50100 + `backend/business-logic/demodata` (YAML data, SMALL/LARGE, run from the Job Queue),
new Country_Region table (page 12) with `Customer.country_region_code`, and `Company.display_name`.
See README "Demo Data".

Remaining:
- [ ] Env flag for docker/CI (e.g. `DEMO_DATA_COMPANY=demo01`, `DEMO_DATA_SIZE=SMALL`): at startup create
      the company if missing and run `demodata.Create` when it has no customers, so Playwright E2E starts
      from known data.
- [ ] Permissions: the READER role (migration 002) has no read permission on `Country_Region`; add it in a
      new migration (also missing for `SMTP_Setup`).
- [ ] Company display name can only be set on the Companies page (11) after creating the company; the
      login page "New company" form takes only the technical name.
- [ ] New business tables (items, sales documents, ...) get demo data in the same change.

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
encrypt masked setup fields (e.g. the SMTP password) at rest — the API already masks them.

---

## Feature: Editable List — BC Record Entry Behavior ✅ IMPLEMENTED

**Status:** implemented. The rules now live in `CLAUDE.md` → *Record Entry (BC/NAV insert lifecycle)*
and *Cross-field validation*. Resolved open questions: a failed INSERT keeps the row uncommitted and
editable (retried on the next confirmed cell); "touched" is a diff against the row's `_pristine` init
values, dropped after insert; the init endpoint takes no filter context yet (see follow-ups); no
validate gating was needed — only new rows call validate, existing rows get trigger results from the
modify response. Work that landed alongside it:
- **Triggers never ran from the API.** `getTable` called the base `InitWithDBType`, so wrapper
  `SetTriggers` never ran (no OnInsert/OnModify/OnDelete), and Go has no virtual dispatch, so wrapper
  `OnValidate_*` overrides were dead code. Wrappers now override `InitWithDBType` and register
  `SetSelf(t)`; generated `ValidateField` dispatches through it. The relation validators in
  `customer.go`/`customerledgerentry.go` now pass the DB type (they hardcoded SQLite placeholders).
- **PK edits were silent no-ops.** Generated `Modify()` never updated primary key columns (an all-PK
  table like `User_Member` returned success without writing). It now renames in place (BC Rename).
- `InsertRecord`/`ModifyRecord` validate only changed fields, in table field order.

Follow-ups (not done):
- [x] Trigger errors surface as a generic "insert/modify failed" — **fixed.** Tables keep the failing
      OnInsert/OnModify/OnDelete error (`TriggerError()`); the API returns it with 400 and the frontend
      shows it (delete toasts too). Database errors keep the generic message.
- [ ] Trigger/validation messages are hardcoded English (`errors.New("name cannot exceed 50
      characters")` in the table wrappers' `Validate()`), and users now see them. Move them to
      `translations/*/errors.yaml` per the CLAUDE.md i18n rule.
- [x] `User.OnDelete` preferences cascade missed — **fixed.** It deletes from the same company-less
      preferences table the handler uses, by exact user ID. Preferences of users deleted before this
      fix are still orphaned in `$User_Preferences`; clean them up once if it matters.
- [ ] Init endpoint filter context — seed a new row from the list's current filters (BC seeds a
      journal line from its batch). Pass filters when the first journal page needs it.
- [ ] Editing a primary key cell of a record loaded from the database: `getRecordKey` keys such
      rows by their PK values (only new rows use `_tempId`), so each keystroke changes the row's key
      and Svelte may re-create the row mid-edit (focus loss). Not yet reproduced — verify, then key
      editable rows by `_key` (their persisted key) instead.
- [ ] Known deviation from BC: picking a value in a `LookupDropdown` saves when focus leaves the cell
      (CLAUDE.md "LookupDropdown Select vs Blur" ABSOLUTE RULE), while BC validates — and inserts —
      on the pick itself (screenshot 04). Kept on purpose; revisit if it matters.
- [x] Manual browser walkthrough (Verification → Manual below) — done by hand against the docker
      compose stack (PostgreSQL).

Original spec, kept for reference — Inserting a row on an editable list page does not match Business
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
- ~~**Security follow-up:** filter field names and `sort_by` interpolated straight into SQL~~ —
  **fixed.** Generated tables map field names through a column allowlist (`columnName`/`HasColumn`)
  in `SetFilter`, `SetRange`, `SetCurrentKey` and `ModifyAll`; an unknown filter field fails closed
  (`1=0`, so a mistyped filter can never widen a `DeleteAll`). The list/ids handlers reject unknown
  fields with 400. `SanitizeFieldName` (regex strip) was not used — an allowlist is stricter.
- [ ] Filtering on FlowFields (e.g. Customer `balance_lcy`): `FilterPane` offers every repeater field,
      but FlowFields are not columns, so the API now returns 400 "Invalid filters parameter" (before:
      an SQL error). BC supports it via CalcFields; needs its own implementation.
- Totals/footer row, grouping, FactBox pane, export to Excel/CSV, "Show as chart", row-level style
  expressions, expand/collapse rows.

---

_Not tracked here (identified as noise, not real TODOs): i18n `%1`/`%2` substitution logic,
SQL `$N`/`?` parameter builders, Svelte input `placeholder=` attributes, UUID templates in
`toast.ts`, and `vi.stubGlobal` test helpers._
