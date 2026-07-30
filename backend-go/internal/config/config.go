package config

import (
	// "log"
	"os"
	"strconv"
	// "time"
	// "github.com/go-sql-driver/mysql"
)

type Config struct {
	DB         DBConfig
	Redis      RedisConfig
	JWTSecret  string
	MaxClients int
	WhatsApp   WhatsAppConfig
}

type WhatsAppConfig struct {
	PhoneNumberID string
	AccessToken   string
	VerifyToken   string
	APIVersion    string
}

type DBConfig struct {
	DSN string
}

type RedisConfig struct {
	URL string
}

func Load() *Config {
	maxClients, _ := strconv.Atoi(getEnv("MAX_CLIENTS", "3"))

	return &Config{
		DB: DBConfig{
			DSN: getEnv("DB_DSN", "saas_user:saas_pass@tcp(localhost:3306)/saas_atendimento?parseTime=true"),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "localhost:6379"),
		},
		JWTSecret:  getEnv("JWT_SECRET", "supersecretkey123"),
		MaxClients: maxClients,
		WhatsApp: WhatsAppConfig{
			PhoneNumberID: getEnv("WHATSAPP_PHONE_ID", ""),
			AccessToken:   getEnv("WHATSAPP_ACCESS_TOKEN", ""),
			VerifyToken:   getEnv("WHATSAPP_VERIFY_TOKEN", "seu_verify_token"),
			APIVersion:    getEnv("WHATSAPP_API_VERSION", "v18.0"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
