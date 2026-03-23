package observability

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger crea un logger estructurado según entorno y nivel.
//
// local       → formato legible por humanos con colores
// production  → formato JSON para ingestión en ELK/Loki/etc.
func NewLogger(level, environment string) (*zap.Logger, error) {
	var lvl zapcore.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", level, err)
	}

	var cfg zap.Config
	if environment == "local" {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
	}

	cfg.Level = zap.NewAtomicLevelAt(lvl)

	logger, err := cfg.Build(
		zap.WithCaller(true),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, fmt.Errorf("building logger: %w", err)
	}

	// Disponible globalmente como zap.L() para librerías que lo usen
	zap.ReplaceGlobals(logger)

	return logger, nil
}
