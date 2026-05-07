package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	ServerPort  string
	Env         string
	DB          DBConfig
	AbacatePay  AbacatePayConfig
	JWT         JWTConfig
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func (d DBConfig) DSN() string {
	if d.SSLMode == "" {
		d.SSLMode = "disable"
	}

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host,
		d.Port,
		d.User,
		d.Password,
		d.DBName,
		d.SSLMode,
	)
}

// AbacatePayConfig configuração da API Abacate Pay
type AbacatePayConfig struct {
	BaseURL        string
	V2BaseURL      string   // API v2 (assinaturas: /subscriptions/*)
	APIKey         string
	WebhookSecret  string   // secret configurado no dashboard para validar webhooks
	PaymentMethods []string // cobrança avulsa (v1 billing/create): PIX, CARD
	SubMethods     []string // checkout de assinatura (v2 subscriptions/create); padrão CARD (doc Abacate Pay)
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
			BaseURL:        getEnv("ABACATE_PAY_BASE_URL", "https://api.abacatepay.com/v1"),
			V2BaseURL:      getEnv("ABACATE_PAY_V2_BASE_URL", "https://api.abacatepay.com/v2"),
			APIKey:         os.Getenv("ABACATE_PAY_API_KEY"),
			WebhookSecret:  os.Getenv("ABACATE_PAY_WEBHOOK_SECRET"),
			PaymentMethods: parsePaymentMethods(getEnv("ABACATE_PAY_METHODS", "PIX")),
			SubMethods:     parsePaymentMethods(getEnv("ABACATE_PAY_SUBSCRIPTION_METHODS", "CARD")),
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

func parsePaymentMethods(s string) []string {
	var out []string
	for _, v := range strings.Split(strings.TrimSpace(s), ",") {
		v = strings.TrimSpace(strings.ToUpper(v))
		if v != "" && (v == "PIX" || v == "CARD") {
			seen := false
			for _, x := range out {
				if x == v {
					seen = true
					break
				}
			}
			if !seen {
				out = append(out, v)
			}
		}
	}
	if len(out) == 0 {
		return []string{"PIX"}
	}
	return out
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
