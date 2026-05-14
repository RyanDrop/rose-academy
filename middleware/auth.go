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
			tokenString := bearerTokenFromRequest(r)
			if tokenString == "" {
				api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "token ausente")
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

// bearerTokenFromRequest extrai o JWT do header Authorization (Bearer ou token cru).
func bearerTokenFromRequest(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	if len(parts) == 1 && parts[0] != "" {
		return parts[0]
	}
	return ""
}

// OptionalAuth valida o Bearer se existir e coloca user_id no context; sem header ou token inválido segue sem usuário (não responde 401).
func OptionalAuth(jwtConfig JWTConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := bearerTokenFromRequest(r)
			if tokenString == "" {
				next.ServeHTTP(w, r)
				return
			}
			userID, err := jwtConfig.ValidateToken(tokenString)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
