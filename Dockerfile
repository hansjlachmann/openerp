# Build stage
FROM golang:1.24-alpine AS builder

# Version passed at build time
ARG VERSION=dev
# Optional: path to extensions directory (for child repos)
ARG EXTENSIONS_PATH=""

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the extmerge tool
RUN CGO_ENABLED=0 GOOS=linux go build -o extmerge ./tools/extmerge

# If EXTENSIONS_PATH is provided, merge extensions into business-logic
RUN if [ -n "$EXTENSIONS_PATH" ] && [ -d "$EXTENSIONS_PATH" ]; then \
      echo "Merging extensions from $EXTENSIONS_PATH..."; \
      mkdir -p /tmp/merged; \
      ./extmerge --core ./backend/business-logic --extensions "$EXTENSIONS_PATH" --output /tmp/merged --verbose; \
      rm -rf ./backend/business-logic; \
      mv /tmp/merged ./backend/business-logic; \
    else \
      echo "No extensions path provided, using core business-logic"; \
    fi

# Build the application with version embedded
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X github.com/hansjlachmann/openerp/backend/api.Version=${VERSION}" -o api-server ./cmd/api-server

# Runtime stage
FROM alpine:3.21

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy binary from builder
COPY --from=builder /app/api-server .

# Copy go.mod as marker for project root detection
COPY --from=builder /app/go.mod ./go.mod

# Copy translation files
COPY --from=builder /app/translations ./translations

# Copy business logic definitions (for table/page YAML files if needed at runtime)
COPY --from=builder /app/backend/business-logic ./backend/business-logic

# Expose port
EXPOSE 8080

# Set environment variables with defaults
ENV PORT=8080
ENV DB_HOST=""
ENV DB_PORT=5432
ENV DB_USER=openerp
ENV DB_PASSWORD=openerp
ENV DB_NAME=openerp
ENV COMPANY_NAME=cronus

# Run the application
CMD ["./api-server"]
