package config

import (
	"os"
	"strconv"
	"time"
)

// Config representa a configuração completa da aplicação
type Config struct {
	DB           DBConfig
	Redis        RedisConfig
	JWTSecret    string
	MaxClients   int
	WhatsApp     WhatsAppConfig
	Server       ServerConfig
	Log          LogConfig
	AI           AIConfig
	Security     SecurityConfig
	Integrations IntegrationsConfig
}

// DBConfig configuração do banco de dados
type DBConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration // em duração
	ConnMaxIdleTime time.Duration // em duração
}

// RedisConfig configuração do Redis
type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

// WhatsAppConfig configuração da API WhatsApp
type WhatsAppConfig struct {
	PhoneNumberID string
	AccessToken   string
	VerifyToken   string
	APIVersion    string
	WebhookURL    string
}

// ServerConfig configuração do servidor HTTP
type ServerConfig struct {
	Port         string
	Mode         string // debug, release, test
	ReadTimeout  int    // em segundos
	WriteTimeout int    // em segundos
	IdleTimeout  int    // em segundos
}

// LogConfig configuração de logging
type LogConfig struct {
	Level  string // debug, info, warn, error
	Format string // json, text
	Output string // stdout, file
}

// AIConfig configuração do serviço de IA
type AIConfig struct {
	URL        string
	Timeout    int // em segundos
	MaxRetries int
	ModelPath  string
	LimiteAuto float64
}

// SecurityConfig configurações de segurança
type SecurityConfig struct {
	JWTSecret      string
	JWTExpiration  int // em horas
	BCryptCost     int
	RateLimit      int // requests por minuto
	AllowedOrigins []string
}

// IntegrationsConfig configurações de integrações
type IntegrationsConfig struct {
	WhatsAppEnabled bool
	IAEnabled       bool
	RedisEnabled    bool
}

// Load carrega as configurações das variáveis de ambiente
func Load() *Config {
	return &Config{
		DB: DBConfig{
			DSN:             getEnv("DB_DSN", "root:root123@tcp(localhost:3306)/saas_atendimento?parseTime=true&charset=utf8mb4&timeout=30s&readTimeout=30s&writeTimeout=30s"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: time.Duration(getEnvAsInt("DB_CONN_MAX_LIFETIME", 30)) * time.Minute,
			ConnMaxIdleTime: time.Duration(getEnvAsInt("DB_CONN_MAX_IDLE_TIME", 5)) * time.Minute,
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWTSecret:  getEnv("JWT_SECRET", "supersecretkey123"),
		MaxClients: getEnvAsInt("MAX_CLIENTS", 3),
		WhatsApp: WhatsAppConfig{
			PhoneNumberID: getEnv("WHATSAPP_PHONE_ID", ""),
			AccessToken:   getEnv("WHATSAPP_ACCESS_TOKEN", ""),
			VerifyToken:   getEnv("WHATSAPP_VERIFY_TOKEN", "seu_verify_token"),
			APIVersion:    getEnv("WHATSAPP_API_VERSION", "v18.0"),
			WebhookURL:    getEnv("WHATSAPP_WEBHOOK_URL", "/api/v1/webhook/whatsapp"),
		},
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", ":8080"),
			Mode:         getEnv("GIN_MODE", "debug"),
			ReadTimeout:  getEnvAsInt("SERVER_READ_TIMEOUT", 30),
			WriteTimeout: getEnvAsInt("SERVER_WRITE_TIMEOUT", 30),
			IdleTimeout:  getEnvAsInt("SERVER_IDLE_TIMEOUT", 60),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
			Output: getEnv("LOG_OUTPUT", "stdout"),
		},
		AI: AIConfig{
			URL:        getEnv("IA_URL", "http://ia-service:8001"),
			Timeout:    getEnvAsInt("IA_TIMEOUT", 30),
			MaxRetries: getEnvAsInt("IA_MAX_RETRIES", 3),
			ModelPath:  getEnv("IA_MODEL_PATH", "/models/llama-2-7b.Q4_K_M.gguf"),
			LimiteAuto: getEnvAsFloat("IA_LIMITE_AUTO", 500.00),
		},
		Security: SecurityConfig{
			JWTSecret:      getEnv("JWT_SECRET", "supersecretkey123"),
			JWTExpiration:  getEnvAsInt("JWT_EXPIRATION", 24),
			BCryptCost:     getEnvAsInt("BCRYPT_COST", 10),
			RateLimit:      getEnvAsInt("RATE_LIMIT", 100),
			AllowedOrigins: getEnvAsSlice("ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:3001"}),
		},
		Integrations: IntegrationsConfig{
			WhatsAppEnabled: getEnvAsBool("WHATSAPP_ENABLED", false),
			IAEnabled:       getEnvAsBool("IA_ENABLED", true),
			RedisEnabled:    getEnvAsBool("REDIS_ENABLED", true),
		},
	}
}

// ============================================
// HELPERS
// ============================================

// getEnv retorna o valor de uma variável de ambiente ou um valor padrão
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt retorna o valor de uma variável de ambiente como inteiro
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvAsFloat retorna o valor de uma variável de ambiente como float64
func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

// getEnvAsBool retorna o valor de uma variável de ambiente como booleano
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// getEnvAsSlice retorna o valor de uma variável de ambiente como slice
func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Dividir por vírgula e remover espaços
		parts := []string{}
		for _, part := range splitByComma(value) {
			if trimmed := trimSpace(part); trimmed != "" {
				parts = append(parts, trimmed)
			}
		}
		if len(parts) > 0 {
			return parts
		}
	}
	return defaultValue
}

// splitByComma divide uma string por vírgula
func splitByComma(s string) []string {
	result := []string{}
	current := ""
	for _, ch := range s {
		if ch == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// trimSpace remove espaços em branco
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// ============================================
// MÉTODOS ÚTEIS
// ============================================

// IsProduction verifica se o ambiente é produção
func (c *Config) IsProduction() bool {
	return c.Server.Mode == "release"
}

// IsDevelopment verifica se o ambiente é desenvolvimento
func (c *Config) IsDevelopment() bool {
	return c.Server.Mode == "debug"
}

// GetDSN retorna a DSN do banco de dados
func (c *Config) GetDSN() string {
	return c.DB.DSN
}

// GetRedisAddr retorna o endereço do Redis
func (c *Config) GetRedisAddr() string {
	return c.Redis.URL
}

// GetJWTSecret retorna o segredo JWT
func (c *Config) GetJWTSecret() string {
	return c.Security.JWTSecret
}

// GetMaxClients retorna o número máximo de clientes em atendimento
func (c *Config) GetMaxClients() int {
	return c.MaxClients
}
