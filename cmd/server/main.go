package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"

	"go.uber.org/zap"

	"github.com/JoX23/go-without-magic/internal/config"
	httphandler "github.com/JoX23/go-without-magic/internal/handler/http"
	"github.com/JoX23/go-without-magic/internal/observability"
	"github.com/JoX23/go-without-magic/internal/repository/memory"
	"github.com/JoX23/go-without-magic/internal/service"
	"github.com/JoX23/go-without-magic/pkg/health"
	"github.com/JoX23/go-without-magic/pkg/shutdown"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal error: %v\n", err)
		os.Exit(1)
	}
}

// run separa la lógica de main para poder retornar errores
// y testear el arranque sin llamar a os.Exit directamente.
func run() error {
	// ── 1. Configuración ───────────────────────────────────────────────
	cfg, err := config.Load("internal/config/config.yaml")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// ── 2. Logger ─────────────────────────────────────────────────────
	logger, err := observability.NewLogger(
		cfg.Observability.LogLevel,
		cfg.Service.Environment,
	)
	if err != nil {
		return fmt.Errorf("creating logger: %w", err)
	}
	defer logger.Sync() //nolint:errcheck

	logger.Info("starting service",
		zap.String("name", cfg.Service.Name),
		zap.String("version", cfg.Service.Version),
		zap.String("environment", cfg.Service.Environment),
	)

	// ── 3. Repositorio ─────────────────────────────────────────────────
	// En local usamos memoria; para producción cambia por postgres.New(cfg.Database)
	repo := memory.NewUserRepository()

	// Para usar PostgreSQL real, reemplaza las líneas anteriores por:
	// repo, err := postgres.New(cfg.Database)
	// if err != nil {
	//     return fmt.Errorf("connecting to database: %w", err)
	// }
	// defer repo.Close()

	// ── 4. Capa de servicio ────────────────────────────────────────────
	userSvc := service.NewUserService(repo, logger)

	// ── 5. HTTP Handler ────────────────────────────────────────────────
	userHandler := httphandler.NewUserHandler(userSvc, logger)

	mux := http.NewServeMux()

	// Rutas de negocio
	userHandler.RegisterRoutes(mux)

	// Rutas de infraestructura
	// Sin checkers reales en modo memoria — agregar repo cuando uses postgres
	mux.Handle("/healthz", health.NewHandler())

	// ── 6. Servidor HTTP ───────────────────────────────────────────────
	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)

	// Chequear que el puerto esté disponible ANTES de crear el servidor
	// Esto nos da error temprano en lugar de una goroutine silenciosa
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("cannot bind to %s: %w", addr, err)
	}

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Canal para reportar errores del servidor
	serverErrors := make(chan error, 1)

	// Arrancar servidor HTTP en goroutine
	// El listener ya está binding al puerto, así que no hay race condition
	go func() {
		logger.Info("HTTP server listening", zap.String("addr", addr))
		// Usar Serve() en lugar de ListenAndServe() porque ya tenemos el listener
		serverErrors <- httpServer.Serve(lis)
	}()

	// ── 7. Graceful Shutdown ───────────────────────────────────────────
	shutdownMgr := shutdown.NewManager(cfg.Server.ShutdownTimeout, logger).
		Register("http", httpServer)

	// Iniciar el signal handler en goroutine (no bloquea)
	go shutdownMgr.Wait()

	// Esperar: error del servidor O finalización del shutdown
	// Si el servidor falla al startup, retornamos el error inmediatamente
	err = <-serverErrors
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("HTTP server error: %w", err)
	}

	return nil
}
