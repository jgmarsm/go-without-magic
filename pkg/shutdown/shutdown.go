package shutdown

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// Server es cualquier componente que soporta apagado graceful.
// Implementado por: *http.Server, grpc.Server (via adapter), etc.
type Server interface {
	Shutdown(ctx context.Context) error
}

// Manager coordina el apagado ordenado de múltiples servidores.
//
// Orden de shutdown (LIFO — Last In, First Out):
//  1. Último registrado se apaga primero
//  2. Típicamente: HTTP → gRPC → Base de datos
type Manager struct {
	timeout time.Duration
	logger  *zap.Logger
	servers []namedServer
}

type namedServer struct {
	name   string
	server Server
}

func NewManager(timeout time.Duration, logger *zap.Logger) *Manager {
	return &Manager{
		timeout: timeout,
		logger:  logger,
	}
}

// Register añade servidores en orden de registro.
// El shutdown ocurre en orden inverso (LIFO).
func (m *Manager) Register(name string, s Server) *Manager {
	m.servers = append(m.servers, namedServer{name: name, server: s})
	return m
}

// Wait bloquea hasta recibir SIGINT o SIGTERM,
// luego ejecuta shutdown ordenado con timeout global.
//
// GARANTÍA: aunque un servidor falle, los demás siguen apagándose.
func (m *Manager) Wait() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	m.logger.Info("shutdown signal received",
		zap.String("signal", sig.String()),
	)

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	// Apagar en orden inverso al registro
	for i := len(m.servers) - 1; i >= 0; i-- {
		ns := m.servers[i]
		m.logger.Info("shutting down", zap.String("server", ns.name))

		if err := ns.server.Shutdown(ctx); err != nil {
			m.logger.Error("shutdown error",
				zap.String("server", ns.name),
				zap.Error(err),
			)
		} else {
			m.logger.Info("shutdown complete", zap.String("server", ns.name))
		}
	}
}
