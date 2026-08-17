# ===========================================
# Build stage
# ===========================================
FROM golang:1.23-alpine AS builder

WORKDIR /src

# Copy go.mod/go.sum first for dependency caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy the full project.
COPY . .

# Create a non-root user entry in /passwd for the scratch image.
RUN echo 'appuser:x:1000:1000::/nonexistent:/usr/sbin/nologin' > /passwd && \
    echo 'appgroup:x:1000:' >> /passwd

# Build a statically linked binary (CGO disabled).
# Do NOT hardcode GOARCH — buildx sets it automatically for multi-arch builds.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/mcp-server .

# ===========================================
# Runtime stage
# ===========================================
FROM scratch

# CA certificates in case the gateway is reached over TLS.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Non-root user passwd entry.
COPY --from=builder /passwd /etc/passwd

# Copy the compiled binary.
COPY --from=builder /out/mcp-server /app/mcp-server

USER 1000:1000

ENV PORT=8080

EXPOSE 8080

ENTRYPOINT ["/app/mcp-server"]
