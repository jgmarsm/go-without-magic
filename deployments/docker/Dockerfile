# ── Stage 1: Build ────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

# Certificados SSL para llamadas HTTPS dentro del binario
# tzdata para manejo correcto de zonas horarias
RUN apk add --no-cache ca-certificates tzdata git

# Usuario sin privilegios para el runtime
RUN adduser -D -u 1001 -g '' appuser

WORKDIR /build

# Cachear dependencias en layer separado
# Solo se re-descarga si cambia go.mod o go.sum
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

# Binary estático sin CGO — compatible con scratch/distroless
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w -extldflags '-static'" \
    -trimpath \
    -o server \
    ./cmd/server

# ── Stage 2: Runtime ─────────────────────────────────────────────────────
# scratch = imagen vacía, tamaño mínimo, superficie de ataque mínima
FROM scratch

# Copiar solo lo estrictamente necesario
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /build/server /server
COPY --from=builder /build/internal/config/config.yaml /internal/config/config.yaml

USER appuser

EXPOSE 8080

# Healthcheck usando el endpoint del propio servicio
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/bin/sh", "-c", "wget -qO- http://localhost:8080/healthz || exit 1"]

ENTRYPOINT ["/server"]