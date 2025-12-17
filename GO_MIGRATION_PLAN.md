# OpenERP Go - Migration Plan

## Overview
Migrate OpenERP from Python to Go+Python hybrid architecture for better performance while maintaining trigger flexibility.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Go Core Application                       │
│                                                              │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │  Database   │  │    CRUD      │  │   Multi-Company  │  │
│  │   Layer     │  │   Manager    │  │     Logic        │  │
│  └─────────────┘  └──────────────┘  └──────────────────┘  │
│                                                              │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │  HTTP API   │  │  NAV-Style   │  │   Concurrency    │  │
│  │   Server    │  │   Records    │  │   (Goroutines)   │  │
│  └─────────────┘  └──────────────┘  └──────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         Python Trigger Engine (Embedded)             │  │
│  │         - RestrictedPython execution                 │  │
│  │         - User-defined business logic                │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Performance Goals

| Operation | Python (Current) | Go+Python (Target) | Improvement |
|-----------|------------------|-------------------|-------------|
| Simple INSERT | ~1ms | 0.1ms | 10x |
| INSERT with trigger | ~2ms | 1.5ms | 1.3x |
| Complex query | 50ms | 5ms | 10x |
| 1000 concurrent requests | Slow | Fast | 50x+ |
| REST API latency | 10ms | 1ms | 10x |

## Phase 1: Project Setup

### 1.1 Go Project Structure
```
openerp-go/
├── cmd/
│   └── openerp/
│       └── main.go              # Application entry point
├── internal/
│   ├── database/
│   │   ├── db.go                # Database connection
│   │   ├── metadata.go          # Table/field metadata
│   │   └── migrations.go        # Schema migrations
│   ├── crud/
│   │   ├── crud.go              # CRUD operations
│   │   └── query.go             # Query builder
│   ├── company/
│   │   ├── company.go           # Company management
│   │   └── resolver.go          # Table name resolution
│   ├── triggers/
│   │   ├── engine.go            # Trigger execution (calls Python)
│   │   └── manager.go           # Trigger management
│   ├── record/
│   │   └── record.go            # NAV-style Record API
│   └── api/
│       ├── handlers.go          # HTTP handlers
│       └── routes.go            # API routes
├── pkg/
│   └── python/
│       ├── executor.go          # Python embedding
│       └── bridge.go            # Go<->Python bridge
├── python/
│   ├── trigger_engine.py        # Python trigger executor
│   └── restricted_runtime.py   # RestrictedPython setup
├── go.mod
├── go.sum
└── README.md
```

### 1.2 Required Go Dependencies
```bash
go get github.com/mattn/go-sqlite3
go get github.com/go-python/gopy/gopyh
go get github.com/gorilla/mux          # For HTTP routing
go get github.com/rs/zerolog           # For logging
```

### 1.3 Python Requirements (Embedded)
```
RestrictedPython>=7.0
```

## Phase 2: Core Implementation

### 2.1 Database Layer (Go)
- SQLite connection with pooling
- Prepared statement caching
- Transaction management
- Table metadata CRUD

### 2.2 Multi-Company Logic (Go)
- Company table management
- Table name resolution (Company$Table)
- Company-specific queries

### 2.3 CRUD Operations (Go)
- Insert (with trigger hooks)
- Update (with trigger hooks)
- Delete (with trigger hooks)
- Query (Get, GetAll, Search)

## Phase 3: Python Integration

### 3.1 Python Embedding
Use `go-python/gopy` or `cgo` to embed Python interpreter:
```go
import "C"
import "unsafe"

func ExecuteTrigger(code string, record map[string]interface{}) (map[string]interface{}, error) {
    // Call Python from Go
    py_code := C.CString(code)
    defer C.free(unsafe.Pointer(py_code))

    result := C.execute_python_trigger(py_code, recordJSON)
    return parseResult(result), nil
}
```

### 3.2 Trigger Bridge
- Go calls Python for trigger execution
- Python returns modified record or error
- Minimal overhead (in-process call)

### 3.3 RestrictedPython Integration
Keep existing Python trigger engine:
```python
# python/trigger_engine.py
from RestrictedPython import compile_restricted, safe_globals

def execute_trigger(code: str, record: dict, old_record: dict = None) -> dict:
    """Execute trigger and return modified record"""
    byte_code = compile_restricted(code, '<trigger>', 'exec')
    exec_globals = safe_globals.copy()
    exec_globals['record'] = record
    exec_globals['old_record'] = old_record
    exec(byte_code, exec_globals)
    return exec_globals['record']
```

## Phase 4: Advanced Features

### 4.1 NAV-Style API (Go)
```go
// NAV-style Record API in Go
customer := NewCustomer(db)
customer.Get(1)
customer.Balance = 2500.0
customer.Modify()
```

### 4.2 HTTP API
- RESTful endpoints
- JSON request/response
- Authentication
- Rate limiting

### 4.3 Concurrency
- Goroutines for parallel operations
- Channel-based communication
- Connection pooling

## Migration Strategy

### Option A: Big Bang (Risky)
Rewrite everything at once, switch completely.

### Option B: Gradual Migration (Recommended)
1. **Week 1**: Build Go core, run alongside Python
2. **Week 2**: Migrate read operations to Go
3. **Week 3**: Migrate write operations to Go
4. **Week 4**: Switch completely, deprecate Python

### Option C: Parallel Development
- Keep Python version running
- Build Go version incrementally
- Test thoroughly before switching

## Testing Plan

### Unit Tests
- Go: `go test ./...`
- Python: Keep existing tests

### Integration Tests
- CRUD operations
- Trigger execution
- Multi-company isolation

### Performance Tests
- Benchmark Go vs Python
- Load testing (concurrent users)
- Profiling (CPU, memory)

## Deployment

### Build
```bash
# Compile Go with embedded Python
CGO_ENABLED=1 go build -o openerp cmd/openerp/main.go
```

### Run
```bash
# Single binary (with Python runtime required)
./openerp --config config.yaml
```

### Docker
```dockerfile
FROM golang:1.21 AS builder
RUN apt-get update && apt-get install -y python3-dev
COPY . /app
WORKDIR /app
RUN go build -o openerp cmd/openerp/main.go

FROM ubuntu:22.04
RUN apt-get update && apt-get install -y python3 python3-pip
COPY --from=builder /app/openerp /usr/local/bin/
COPY python/ /app/python/
RUN pip3 install RestrictedPython
CMD ["/usr/local/bin/openerp"]
```

## Risks & Mitigation

| Risk | Mitigation |
|------|-----------|
| CGO complexity | Use well-maintained libraries (go-python/gopy) |
| Python GIL bottleneck | Profile, optimize hot paths |
| Breaking changes | Comprehensive testing, gradual migration |
| Developer learning curve | Documentation, examples, pair programming |

## Success Criteria

- [ ] 10x performance improvement on database operations
- [ ] All existing triggers work unchanged
- [ ] NAV-style API implemented
- [ ] All tests passing
- [ ] Production-ready deployment

## Next Steps

1. Set up Go project (today)
2. Implement database layer (this week)
3. Proof-of-concept Python embedding (this week)
4. Performance benchmarks (next week)
5. Full migration (2-4 weeks)
