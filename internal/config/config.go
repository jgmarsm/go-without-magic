package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config contiene toda la configuración del servicio.
// Todos los campos son tipados — nunca map[string]interface{}.
type Config struct {
	Service       ServiceConfig       `mapstructure:"service"`
	Server        ServerConfig        `mapstructure:"server"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Observability ObservabilityConfig `mapstructure:"observability"`
}

type ServiceConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
}

type ServerConfig struct {
	HTTPPort        int           `mapstructure:"http_port"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	DSN          string `mapstructure:"dsn"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type ObservabilityConfig struct {
	LogLevel string `mapstructure:"log_level"`
}

// Load carga configuración desde archivo YAML + variables de entorno.
//
// Las variables de entorno sobreescriben el archivo YAML.
// Formato: APP_SERVER_HTTP_PORT=9090 → server.http_port
func Load(path string) (*Config, error) {
	v := viper.New()

	// Valores por defecto seguros
	setDefaults(v)

	// Archivo de configuración
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// Variables de entorno: APP_SERVER_HTTP_PORT → server.http_port
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// No es error fatal si el archivo no existe (usamos defaults + env)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config file %q: %w", path, err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// validate verifica que los campos críticos estén presentes.
func (c *Config) validate() error {
	if c.Service.Name == "" {
		return fmt.Errorf("service.name is required")
	}
	if c.Server.HTTPPort == 0 {
		return fmt.Errorf("server.http_port is required")
	}
	return nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("service.version", "dev")
	v.SetDefault("service.environment", "local")
	v.SetDefault("server.http_port", 8080)
	v.SetDefault("server.shutdown_timeout", 30*time.Second)
	v.SetDefault("server.read_timeout", 10*time.Second)
	v.SetDefault("server.write_timeout", 10*time.Second)
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("observability.log_level", "info")
}
