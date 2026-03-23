package shutdown

import (
	"context"
	"os"
	"os/signal"
	"sync"
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
//
// CONCURRENCIA:
// - Wait() es seguro para llamar desde múltiples goroutines simultáneamente
// - Solo ejecuta el shutdown UNA sola vez (sync.Once)
// - Todos los Wait() se bloquean hasta que shutdown es completo
type Manager struct {
	timeout   time.Duration
	logger    *zap.Logger
	servers   []namedServer
	once      sync.Once     // ← Garantiza una sola ejecución
	done      chan struct{} // ← Señal de finalización
	startOnce sync.Once     // ← Inicializar done una sola vez
}

type namedServer struct {
	name   string
	server Server
}

func NewManager(timeout time.Duration, logger *zap.Logger) *Manager {
	return &Manager{
		timeout: timeout,
		logger:  logger,
		done:    make(chan struct{}),
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
// GARANTÍAS:
// 1. Aunque se llame desde múltiples goroutines, shutdown ocurre UNA sola vez
// 2. Aunque un servidor falle, los demás siguen apagándose
// 3. Thread-safe para múltiples Wait() concurrentes
//
// EJEMPLO:
//
//	m := shutdown.NewManager(30*time.Second, logger)
//	m.Register("http", httpServer).Register("database", dbServer)
//
//	go m.Wait()  // Goroutine A
//	go m.Wait()  // Goroutine B (seguro - Wait() es idempotente)
//
// Ambas se bloquean hasta que se reciba la señal de shutdown.
func (m *Manager) Wait() {
	// Convertir a un canal de señales una sola vez
	m.startOnce.Do(func() {
		go m.listenAndShutdown()
	})

	// Bloquear hasta que shutdown complete
	<-m.done
}

// listenAndShutdown es la goroutine que realmente maneja el apagado.
// Se ejecuta UNA sola vez gracias a sync.Once.
func (m *Manager) listenAndShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	m.logger.Info("shutdown signal received",
		zap.String("signal", sig.String()),
	)

	// Ejecutar el shutdown real dentro de sync.Once para evitar duplicación
	m.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
		defer cancel()

		// Apagar en orden inverso al registro (LIFO)
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

		// Señalar a todos los Wait() que pueden retornar
		close(m.done)
	})
}
