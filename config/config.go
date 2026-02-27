package config

import (
	"fmt"
	"os"
	"time"
)

// Config armazena as configurações da aplicação
type Config struct {
	ServerPort  string
	Env         string
	DB          DBConfig
	AbacatePay  AbacatePayConfig
	JWT         JWTConfig
}

// DBConfig configuração do PostgreSQL
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DSN retorna a connection string do PostgreSQL
func (d DBConfig) DSN() string {
	if d.SSLMode == "" {
		d.SSLMode = "disable"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode)
}

// AbacatePayConfig configuração da API Abacate Pay
type AbacatePayConfig struct {
	BaseURL string
	APIKey  string
}

// JWTConfig configuração do JWT
type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

// Load carrega as configurações a partir de variáveis de ambiente
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}
	jwtExpHours := 72
	if e := os.Getenv("JWT_EXPIRATION_HOURS"); e != "" {
		if h, err := parseInt(e); err == nil && h > 0 {
			jwtExpHours = h
		}
	}
	return &Config{
		ServerPort: port,
		Env:        env,
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "admin"),
			Password: getEnv("DB_PASSWORD", "password"),
			DBName:   getEnv("DB_NAME", "rose_gym_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		AbacatePay: AbacatePayConfig{
			BaseURL: getEnv("ABACATE_PAY_BASE_URL", "https://api.abacatepay.com/v1"),
			APIKey:  os.Getenv("ABACATE_PAY_API_KEY"),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "change-me-in-production"),
			Expiration: time.Duration(jwtExpHours) * time.Hour,
		},
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
