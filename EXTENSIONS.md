# OpenERP Extensions

This document describes how to extend OpenERP with custom tables, pages, and codeunits.

## ID Range Conventions

To avoid conflicts between core and extensions, the following ID ranges are reserved:

| Type | Core Range | Extension Range |
|------|------------|-----------------|
| Tables | 1 - 49,999 | 50,000 - 99,999 |
| Pages | 1 - 49,999 | 50,000 - 99,999 |
| Codeunits | 1 - 49,999 | 50,000 - 99,999 |

### Core IDs (1 - 49,999)

Reserved for the base OpenERP application. These IDs are managed in this repository and should not be used by extensions.

### Extension IDs (50,000 - 99,999)

Available for customer-specific customizations. Each customer/extension should claim a sub-range to avoid conflicts:

| Customer | Tables | Pages | Codeunits |
|----------|--------|-------|-----------|
| Customer A | 50,000 - 50,999 | 50,000 - 50,999 | 50,000 - 50,999 |
| Customer B | 51,000 - 51,999 | 51,000 - 51,999 | 51,000 - 51,999 |
| ... | ... | ... | ... |

## Extension Types

### 1. New Objects

Create entirely new tables, pages, or codeunits using IDs in the extension range:

```yaml
# 50000-custom-table.yaml
table:
  id: 50000
  name: "Custom_Inventory"
  # ... full table definition
```

### 2. Extensions to Existing Objects

Extend core objects using `.extend.yaml` files:

```yaml
# 21-customer-card.extend.yaml
extends: 21

actions:
  - name: Custom Report
    caption: Custom Report
    run_object: codeunit:50100
    promoted: true

layout:
  sections:
    - name: General          # Add to existing section
      fields:
        - source: custom_field
          caption: Custom Field
          editable: true

    - name: Custom Section   # Add new section
      caption: Custom Data
      fields:
        - source: custom_data
          editable: true
```

## Extension File Format

### Page Extension

```yaml
extends: <page_id>           # Required: ID of parent page to extend
version: ">=0.1.9"           # Optional: Compatible core versions

# Add actions (appended to existing actions)
actions:
  - name: Action Name
    caption: Display Caption
    shortcut: Ctrl+X         # Optional
    promoted: true           # Optional
    run_object: codeunit:ID  # Or run_page: ID

# Extend layout
layout:
  sections:
    - name: Existing Section # Merge into existing section
      fields:
        - source: field_name
          caption: Caption
          editable: true

    - name: New Section      # Add new section
      caption: Section Caption
      fields:
        - source: field_name
```

### Table Extension

```yaml
extends_table: <table_id>    # Required: ID of parent table to extend

fields:
  - name: custom_field
    type: types.Text
    length: 100
    caption: Custom Field
```

## Merge Rules

| Element | Behavior |
|---------|----------|
| `actions` | Appended to parent's action list |
| `layout.sections` (existing name) | Fields merged into that section |
| `layout.sections` (new name) | Added as new section |
| `fields` (same source) | Overrides parent field definition |
| Other properties | Overrides parent value |

## Build Process

Extensions are merged at build time using the `extmerge` tool:

```bash
# Merge extensions with core
extmerge --core ./backend/definitions \
         --extensions ./extensions \
         --output ./merged

# then generate the table code from the merged definitions
go generate ./backend/generated/tables

# Or in Docker build (merges, regenerates and builds)
docker build --build-arg EXTENSIONS_PATH=./extensions .
```

## Directory Structure (Extension Repository)

```
my-openerp-extension/
├── extensions/
│   ├── tables/
│   │   └── 18-customer.extend.yaml      # Extend Customer table
│   ├── pages/
│   │   └── 21-customer-card.extend.yaml # Extend Customer Card
│   └── logic/                           # Go module: business logic (see below)
│       ├── customer.go                  # Triggers for the Customer fields
│       └── 50100-custom-report.go       # New codeunit
├── custom/
│   ├── tables/
│   │   └── 50000-inventory.yaml         # New table
│   └── pages/
│       └── 50001-inventory-list.yaml    # New page
├── openerp.yaml                          # Core version config
└── .github/workflows/build.yml
```

## openerp.yaml (Extension Config)

```yaml
core:
  repository: hansjlachmann/openerp
  version: v0.1.9

customer:
  name: my-company
  registry: ghcr.io/my-company

id_ranges:
  tables: 50000-50999
  pages: 50000-50999
  codeunits: 50000-50999
```

## Extension apps: business logic (planned)

The base application is built so that an extension app can add business logic without
changing the base code. What already works in the base:

- **Generated code is derived from the merged definitions.** `backend/generated/tables` is
  regenerated after the merge, so a field from `customer.extend.yaml` becomes a typed field
  of `gtables.Customer` and gets its own `OnValidate_<Field>` trigger.
- **Business logic is a list per table.** The base application registers its triggers in
  `backend/business-logic/tables` (`gtables.RegisterCustomerLogic(customerLogic{})`). An
  extension registers its own logic type for the same table, and every registered logic runs
  in registration order (base first). The first error stops the operation.

```go
// my-openerp-extension/extensions/logic/customer.go
package logic

import gtables "github.com/hansjlachmann/openerp/backend/generated/tables"

type customerLogic struct{}

func init() { gtables.RegisterCustomerLogic(customerLogic{}) }

// Runs after the base application's Customer OnInsert
func (customerLogic) OnInsert(rec *gtables.Customer) error {
	if rec.Loyalty_points == 0 {
		rec.Loyalty_points = 100
	}
	return nil
}

func (customerLogic) OnValidate_Loyalty_points(rec *gtables.Customer) error { … }
```

Still to build (`TODO.md`, "Extension apps"):

- **Linking the extension's Go module into the build:** a `go.work` file or a `replace`
  directive pointing at the merged, regenerated base, plus a generated file with a blank
  import per installed extension.
- **A lossless page merge in `extmerge`:** today it drops page keys it does not know.
- **Field-name collision rules.**
- **OnBefore/OnAfter and codeunit events.**
