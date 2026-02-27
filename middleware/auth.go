package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"rose-academy/api"
)

type contextKey string

const UserIDContextKey contextKey = "user_id"

// JWTClaims claims do token
type JWTClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// JWTConfig configuração para gerar e validar JWT
type JWTConfig struct {
	Secret     []byte
	Expiration time.Duration
}

// NewJWTConfig cria config a partir de secret string e duração
func NewJWTConfig(secret string, expiration time.Duration) JWTConfig {
	return JWTConfig{
		Secret:     []byte(secret),
		Expiration: expiration,
	}
}

// GenerateToken gera um JWT para o userID
func (c JWTConfig) GenerateToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(c.Expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   userID.String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(c.Secret)
}

// ValidateToken retorna o userID se o token for válido
func (c JWTConfig) ValidateToken(tokenString string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		return c.Secret, nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}
	id, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// Auth middleware que exige Authorization: Bearer <token> e coloca user_id no context
func Auth(jwtConfig JWTConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := strings.TrimSpace(r.Header.Get("Authorization"))
			if auth == "" {
				api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "token ausente")
				return
			}
			tokenString := ""
			parts := strings.SplitN(auth, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				tokenString = strings.TrimSpace(parts[1])
			} else if len(parts) == 1 && parts[0] != "" {
				tokenString = parts[0]
			}
			if tokenString == "" {
				api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "envie o token no header: Authorization: Bearer <token>")
				return
			}
			userID, err := jwtConfig.ValidateToken(tokenString)
			if err != nil {
				api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "token inválido ou expirado")
				return
			}
			ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID obtém o user_id do context (após middleware Auth)
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(UserIDContextKey)
	if v == nil {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}
