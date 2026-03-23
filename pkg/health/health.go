package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Checker es cualquier dependencia que puede verificar su conectividad.
// Implementado por: postgres.UserRepository, redis.Client, etc.
type Checker interface {
	Ping(ctx context.Context) error
}

// NamedChecker asocia un nombre legible a cada checker.
type NamedChecker struct {
	Name    string
	Checker Checker
}

type response struct {
	Status string            `json:"status"`           // "ok" | "degraded"
	Checks map[string]string `json:"checks,omitempty"` // nombre → "ok" | "fail: ..."
}

// NewHandler retorna un http.Handler para GET /healthz.
//
// Responde:
//   - 200 si todos los checkers responden correctamente
//   - 503 si alguno falla (para que Kubernetes/load balancer lo detecte)
func NewHandler(checkers ...NamedChecker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		resp := response{
			Status: "ok",
			Checks: make(map[string]string, len(checkers)),
		}
		statusCode := http.StatusOK

		for _, nc := range checkers {
			if err := nc.Checker.Ping(ctx); err != nil {
				resp.Checks[nc.Name] = fmt.Sprintf("fail: %v", err)
				resp.Status = "degraded"
				statusCode = http.StatusServiceUnavailable
			} else {
				resp.Checks[nc.Name] = "ok"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	})
}
