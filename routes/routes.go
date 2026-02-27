package routes

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"

	"rose-academy/api"
	"rose-academy/config"
	"rose-academy/controllers"
	"rose-academy/models"
	authmw "rose-academy/middleware"
	"rose-academy/services"
)

// Setup configura e retorna o router HTTP
func Setup(cfg *config.Config, database *sql.DB) http.Handler {
	r := chi.NewRouter()

	r.Use(api.Recover)
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)

	// Repositórios
	userRepo := models.NewPostgresUserRepository(database)
	goalRepo := models.NewPostgresGoalRepository(database)
	abpRepo := models.NewPostgresABPCustomerRepository(database)

	// Serviços
	abpClient := services.NewAbacatePayClient(cfg.AbacatePay.BaseURL, cfg.AbacatePay.APIKey)
	authService := services.NewAuthService(database, userRepo, goalRepo, abpRepo, abpClient)

	// JWT
	jwtConfig := authmw.NewJWTConfig(cfg.JWT.Secret, cfg.JWT.Expiration)

	// Controllers
	authCtrl := controllers.NewAuthController(authService, userRepo, jwtConfig, cfg.Env == "development")
	goalsCtrl := controllers.NewGoalsController(goalRepo, cfg.Env == "development")

	// Swagger UI (documentação da API no browser: /swagger/index.html)
	r.Mount("/swagger", httpSwagger.WrapHandler)

	// Rotas públicas
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/goals", goalsCtrl.List)
	r.Post("/auth/register", authCtrl.Register)
	r.Post("/auth/login", authCtrl.Login)

	// Rotas protegidas (exigem Authorization: Bearer <token>)
	r.Group(func(r chi.Router) {
		r.Use(authmw.Auth(jwtConfig))
		r.Get("/auth/me", authCtrl.Me)
	})

	return r
}
