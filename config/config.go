package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	defaultEnv         = "local"
	testEnv            = "test"
	maxPort            = 65536
	minJWTSecretLength = 32
)

type Config[T any] struct {
	Base    `mapstructure:",squash"`
	Service T `mapstructure:"service"`
}

type Base struct {
	App         AppConfig         `yaml:"app" mapstructure:"app"`
	Server      ServerConfig      `yaml:"server" mapstructure:"server"`
	Database    DatabaseConfig    `yaml:"database" mapstructure:"database"`
	Redis       RedisConfig       `yaml:"redis" mapstructure:"redis"`
	JWT         JWTConfig         `yaml:"jwt" mapstructure:"jwt"`
	Tracing     TracingConfig     `yaml:"tracing" mapstructure:"tracing"`
	Logger      LoggerConfig      `yaml:"logger" mapstructure:"logger"`
	GRPC        GRPCConfig        `yaml:"grpc" mapstructure:"grpc"`
	NATS        NATSConfig        `yaml:"nats" mapstructure:"nats"`
	Asynq       AsynqConfig       `yaml:"asynq" mapstructure:"asynq"`
	Mail        MailConfig        `yaml:"mail" mapstructure:"mail"`
	GRPCClients GRPCClientsConfig `yaml:"grpc_clients" mapstructure:"grpc_clients"`
}

type AppConfig struct {
	Name        string `yaml:"name" mapstructure:"name"`
	Environment string `yaml:"environment" mapstructure:"environment"`
	Debug       bool   `yaml:"debug" mapstructure:"debug"`
	Version     string `yaml:"version" mapstructure:"version"`
}

type ServerConfig struct {
	Host            string   `yaml:"host" mapstructure:"host"`
	Port            int      `yaml:"port" mapstructure:"port"`
	ReadTimeout     int      `yaml:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout    int      `yaml:"write_timeout" mapstructure:"write_timeout"`
	ShutdownTimeout int      `yaml:"shutdown_timeout" mapstructure:"shutdown_timeout"`
	AllowedOrigins  []string `yaml:"allowed_origins" mapstructure:"allowed_origins"`
}

type DatabaseConfig struct {
	Host        string `yaml:"host" mapstructure:"host"`
	Port        int    `yaml:"port" mapstructure:"port"`
	Username    string `yaml:"username" mapstructure:"username"`
	Password    string `yaml:"password" mapstructure:"password"`
	Database    string `yaml:"database" mapstructure:"database"`
	SSLMode     string `yaml:"ssl_mode" mapstructure:"ssl_mode"`
	MaxConns    int    `yaml:"max_conns" mapstructure:"max_conns"`
	IdleConns   int    `yaml:"idle_conns" mapstructure:"idle_conns"`
	MaxLifetime int    `yaml:"max_lifetime" mapstructure:"max_lifetime"`
}

type RedisConfig struct {
	Host     string `yaml:"host" mapstructure:"host"`
	Port     int    `yaml:"port" mapstructure:"port"`
	Password string `yaml:"password" mapstructure:"password"`
	DB       int    `yaml:"db" mapstructure:"db"`
	PoolSize int    `yaml:"pool_size" mapstructure:"pool_size"`
}

type JWTConfig struct {
	Secret     string `yaml:"secret" mapstructure:"secret"`
	Expiration int    `yaml:"expiration" mapstructure:"expiration"`
	Issuer     string `yaml:"issuer" mapstructure:"issuer"`
}

type TracingConfig struct {
	Enabled      bool     `yaml:"enabled" mapstructure:"enabled"`
	ServiceName  string   `yaml:"service_name" mapstructure:"service_name"`
	SampleRate   float64  `yaml:"sample_rate" mapstructure:"sample_rate"`
	ExcludePaths []string `yaml:"exclude_paths" mapstructure:"exclude_paths"`
	Exporter     string   `yaml:"exporter" mapstructure:"exporter"`
	OTLPEndpoint string   `yaml:"otlp_endpoint" mapstructure:"otlp_endpoint"`
}

type LoggerConfig struct {
	Level  string `yaml:"level" mapstructure:"level"`
	Format string `yaml:"format" mapstructure:"format"`
}

type GRPCConfig struct {
	Server    GRPCServerConfig    `yaml:"server" mapstructure:"server"`
	Client    GRPCClientConfig    `yaml:"client" mapstructure:"client"`
	RateLimit GRPCRateLimitConfig `yaml:"rate_limit" mapstructure:"rate_limit"`
}

type GRPCServerConfig struct {
	Port            int           `yaml:"port" mapstructure:"port"`
	MaxConnAge      time.Duration `yaml:"max_conn_age" mapstructure:"max_conn_age"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" mapstructure:"shutdown_timeout"`
	RequestTimeout  time.Duration `yaml:"request_timeout" mapstructure:"request_timeout"`
}

type GRPCClientConfig struct {
	DialTimeout      time.Duration `yaml:"dial_timeout" mapstructure:"dial_timeout"`
	KeepAliveTime    time.Duration `yaml:"keepalive_time" mapstructure:"keepalive_time"`
	KeepAliveTimeout time.Duration `yaml:"keepalive_timeout" mapstructure:"keepalive_timeout"`
}

type GRPCRateLimitConfig struct {
	RequestsPerSecond int `yaml:"requests_per_second" mapstructure:"requests_per_second"`
	BurstSize         int `yaml:"burst_size" mapstructure:"burst_size"`
}

type NATSConfig struct {
	URL     string `yaml:"url" mapstructure:"url"`
	Enabled bool   `yaml:"enabled" mapstructure:"enabled"`
}

type AsynqConfig struct {
	Enabled     bool `yaml:"enabled" mapstructure:"enabled"`
	Concurrency int  `yaml:"concurrency" mapstructure:"concurrency"`
}

type MailConfig struct {
	Provider string `yaml:"provider" mapstructure:"provider"`
	SMTPHost string `yaml:"smtp_host" mapstructure:"smtp_host"`
	SMTPPort int    `yaml:"smtp_port" mapstructure:"smtp_port"`
	From     string `yaml:"from" mapstructure:"from"`
	Username string `yaml:"username" mapstructure:"username"`
	Password string `yaml:"password" mapstructure:"password"`
}

type GRPCClientsConfig map[string]GRPCClientTarget

type GRPCClientTarget struct {
	Target string `yaml:"target" mapstructure:"target"`
}

func Load[T any](serviceName string) (*Config[T], error) {
	return LoadFrom[T](serviceName, ".")
}

func LoadFrom[T any](serviceName, baseDir string) (*Config[T], error) {
	env := getEnv("APP_ENV", defaultEnv)

	if env == defaultEnv || env == testEnv {
		_ = godotenv.Load(".env")
		_ = godotenv.Load("config/.env")
	}

	v := viper.New()
	v.SetConfigName(serviceName)
	v.SetConfigType("yaml")
	v.AddConfigPath(fmt.Sprintf("%s/config/%s/", baseDir, env))
	v.AddConfigPath(fmt.Sprintf("config/%s/", env))
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	bindSecrets(v)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg Config[T]
	if err := v.Unmarshal(&cfg, decoderConfigOption()); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := validate(&cfg.Base); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return &cfg, nil
}

func bindSecrets(v *viper.Viper) {
	_ = v.BindEnv("database.host", "DB_HOST")
	_ = v.BindEnv("database.port", "DB_PORT")
	_ = v.BindEnv("database.username", "DB_USERNAME")
	_ = v.BindEnv("database.password", "DB_PASSWORD")
	_ = v.BindEnv("database.database", "DB_DATABASE")
	_ = v.BindEnv("database.ssl_mode", "DB_SSL_MODE")
	_ = v.BindEnv("redis.host", "REDIS_HOST")
	_ = v.BindEnv("redis.port", "REDIS_PORT")
	_ = v.BindEnv("redis.password", "REDIS_PASSWORD")
	_ = v.BindEnv("jwt.secret", "JWT_SECRET")
	_ = v.BindEnv("nats.url", "NATS_URL")
	_ = v.BindEnv("tracing.otlp_endpoint", "TRACING_OTLP_ENDPOINT")
	_ = v.BindEnv("mail.password", "MAIL_PASSWORD")
}

func validate(cfg *Base) error {
	if cfg.Server.Port < 0 || cfg.Server.Port >= maxPort {
		return fmt.Errorf("server.port must be between 0 and 65535, got %d", cfg.Server.Port)
	}
	if cfg.Tracing.SampleRate < 0.0 || cfg.Tracing.SampleRate > 1.0 {
		return fmt.Errorf("tracing.sample_rate must be between 0.0 and 1.0, got %f", cfg.Tracing.SampleRate)
	}
	if cfg.App.Environment != defaultEnv && cfg.App.Environment != testEnv {
		if cfg.JWT.Secret != "" && len(cfg.JWT.Secret) < minJWTSecretLength {
			return fmt.Errorf("JWT_SECRET must be at least 32 characters in non-local environments")
		}
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func decoderConfigOption() viper.DecoderConfigOption {
	return func(dc *mapstructure.DecoderConfig) {
		dc.DecodeHook = mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			trimStringSliceHookFunc(),
		)
	}
}

func trimStringSliceHookFunc() mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		if to != reflect.TypeOf([]string{}) {
			return data, nil
		}
		switch v := data.(type) {
		case []string:
			trimmed := make([]string, 0, len(v))
			for _, s := range v {
				s = strings.TrimSpace(s)
				if s != "" {
					trimmed = append(trimmed, s)
				}
			}
			return trimmed, nil
		case []interface{}:
			trimmed := make([]string, 0, len(v))
			for _, item := range v {
				if s, ok := item.(string); ok {
					s = strings.TrimSpace(s)
					if s != "" {
						trimmed = append(trimmed, s)
					}
				}
			}
			return trimmed, nil
		}
		return data, nil
	}
}
