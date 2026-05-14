package routes

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	chimw "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"

	"rose-academy/api"
	"rose-academy/config"
	"rose-academy/controllers"
	"rose-academy/models"
	authmw "rose-academy/middleware"
	"rose-academy/services"
)

// HealthResponse payload GET /health
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// Health liveness probe
// @Summary		Health check
// @Tags			system
// @Produce		json
// @Success		200	{object}	HealthResponse
// @Router		/health [get]
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// Setup configura e retorna o router HTTP
func Setup(cfg *config.Config, database *sql.DB) http.Handler {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           86400,
	}))

	r.Use(api.Recover)
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)

	// Repositórios
	userRepo := models.NewPostgresUserRepository(database)
	goalRepo := models.NewPostgresGoalRepository(database)
	trainingPlanRepo := models.NewPostgresTrainingPlanRepository(database)
	abpRepo := models.NewPostgresABPCustomerRepository(database)
	productRepo := models.NewPostgresProductRepository(database)
	subRepo := models.NewPostgresUserSubscriptionRepository(database)
	userTrainingRepo := models.NewPostgresUserTrainingRepository(database)
	workoutRepo := models.NewPostgresTrainingWorkoutRepository(database)
	exerciseRepo := models.NewPostgresExerciseRepository(database)
	progressRepo := models.NewPostgresTrainingProgressRepository(database)

	// Serviços
	abpClient := services.NewAbacatePayClient(cfg.AbacatePay.BaseURL, cfg.AbacatePay.V2BaseURL, cfg.AbacatePay.APIKey, cfg.AbacatePay.PaymentMethods, cfg.AbacatePay.SubMethods)
	authService := services.NewAuthService(database, userRepo, goalRepo, abpRepo, abpClient)
	billingService := services.NewBillingService(abpClient, userRepo, abpRepo, productRepo, subRepo)

	// JWT
	jwtConfig := authmw.NewJWTConfig(cfg.JWT.Secret, cfg.JWT.Expiration)

	// Controllers
	authCtrl := controllers.NewAuthController(authService, userRepo, goalRepo, jwtConfig, cfg.Env == "development")
	goalsCtrl := controllers.NewGoalsController(goalRepo, cfg.Env == "development")
	trainingPlansCtrl := controllers.NewTrainingPlansController(trainingPlanRepo, userTrainingRepo, cfg.Env == "development")
	productsCtrl := controllers.NewProductsController(productRepo, cfg.Env == "development")
	billingCtrl := controllers.NewBillingController(billingService, cfg.AbacatePay.WebhookSecret, cfg.Env == "development")
	abpSubCtrl := controllers.NewAbacateSubscriptionsController(billingService, cfg.Env == "development")
	userTrainingCtrl := controllers.NewUserTrainingController(userTrainingRepo, cfg.Env == "development")
	workoutsCtrl := controllers.NewTrainingWorkoutsController(workoutRepo, cfg.Env == "development")
	exercisesCtrl := controllers.NewExercisesController(exerciseRepo, cfg.Env == "development")
	progressCtrl := controllers.NewTrainingProgressController(progressRepo, workoutRepo, cfg.Env == "development")

	// Swagger UI (documentação da API no browser: /swagger/index.html)
	r.Mount("/swagger", httpSwagger.WrapHandler)

	// Rotas públicas
	r.Get("/health", Health)
	r.Get("/goals", goalsCtrl.List)
	r.With(authmw.OptionalAuth(jwtConfig)).Get("/training-plans", trainingPlansCtrl.List)
	r.Get("/exercises", exercisesCtrl.List)
	r.Post("/auth/register", authCtrl.Register)
	r.Post("/auth/login", authCtrl.Login)
	r.Post("/webhooks/abacatepay", billingCtrl.WebhookAbacatePay)

	r.Group(func(r chi.Router) {
		r.Use(authmw.Auth(jwtConfig))
		r.Get("/auth/me", authCtrl.Me)
		r.Patch("/auth/me", authCtrl.PatchMe)
		r.Get("/products", productsCtrl.List)
		r.Post("/subscribe", billingCtrl.Subscribe)
		r.Get("/payments/pix/check", billingCtrl.CheckPixStatus)
		r.Post("/abacate/subscriptions/checkout", abpSubCtrl.CreateCheckout)
		r.Get("/abacate/subscriptions", abpSubCtrl.List)
		r.Post("/abacate/subscriptions/cancel", abpSubCtrl.Cancel)
		r.Get("/subscriptions/active", abpSubCtrl.SubscriptionActive)
		r.Post("/me/training/plan", userTrainingCtrl.SelectPlan)
		r.Get("/me/training/plan", userTrainingCtrl.GetActivePlan)
		r.Get("/me/training/months", userTrainingCtrl.ListPlanMonths)
		r.Post("/me/training/months/{month_number}/start", userTrainingCtrl.StartMonth)
		r.Get("/me/training/months/{month_id}/workouts", workoutsCtrl.ListMonthWorkouts)
		r.Get("/me/training/months/{month_id}/weight-progression", progressCtrl.GetWeightProgression)
		r.Get("/me/training/months/{month_id}/weekly-performance", progressCtrl.GetWeeklyBestPerformance)
		r.Get("/me/training/months/{month_id}/progression-summary", progressCtrl.GetMonthlyProgressionSummary)
		r.Get("/me/training/months/{month_id}/statistics", progressCtrl.GetMonthStatistics)
		r.Post("/me/training/months/{month_id}/progress/reset", progressCtrl.ResetMonthProgress)
		r.Post("/me/training/workouts/{workout_id}/complete", progressCtrl.CompleteWorkout)
		r.Get("/me/training/workouts/day/{day_number}", workoutsCtrl.GetWorkoutOfDay)
		r.Get("/me/training/workouts/{workout_id}", workoutsCtrl.GetWorkout)
		r.Post("/me/training/workout-completions/{completion_id}/sets/batch", progressCtrl.LogExerciseSetsBatch)
		r.Post("/me/training/workout-completions/{completion_id}/sets", progressCtrl.LogExerciseSet)
	})

	return r
}
