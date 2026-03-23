.PHONY: run build test test-cover lint tidy docker-build clean help

# ── Variables ─────────────────────────────────────────────────────────────
SERVICE   := go-without-magic
BUILD_DIR := bin
MAIN      := ./cmd/server

# ── Targets ───────────────────────────────────────────────────────────────

## run: Ejecuta el servidor en modo desarrollo
run:
	go run $(MAIN)

## build: Compila el binario para Linux
build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux go build \
		-ldflags="-s -w" \
		-o $(BUILD_DIR)/$(SERVICE) \
		$(MAIN)
	@echo "Binary: $(BUILD_DIR)/$(SERVICE)"

## test: Ejecuta todos los tests con race detector
test:
	go test -race -count=1 -timeout=60s ./...

## test-cover: Tests + reporte de cobertura en HTML
test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Report: open coverage.html"

## lint: Ejecuta golangci-lint
lint:
	golangci-lint run --timeout=3m ./...

## tidy: Limpia y verifica dependencias
tidy:
	go mod tidy
	go mod verify

## docker-build: Construye la imagen Docker
docker-build:
	docker build \
		-f deployments/docker/Dockerfile \
		-t $(SERVICE):latest \
		.

## clean: Elimina artefactos generados
clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

## help: Muestra esta ayuda
help:
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/## //'