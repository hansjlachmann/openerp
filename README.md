
#####  Individual commands:
```
  Build CLI:
  go build -o openerp-cli main.go

  Build GUI:
  go build -o openerp-gui ./src/gui/

  Run CLI (without building):
  go run main.go

  Run GUI (without building):
  go run ./src/gui/

  Clean binaries:
  rm openerp-cli openerp-gui

  Format code:
  go fmt ./...

  Check code:
  go vet ./...

  Update dependencies:
  go mod tidy
```

