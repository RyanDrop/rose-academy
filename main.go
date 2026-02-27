package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"rose-academy/config"
	"rose-academy/db"
	_ "rose-academy/docs" // gerado por swag init
	"rose-academy/routes"
)

// @title			Rose Academy API
// @version		1.0
// @description	API da Rose Academy: auth, metas e integração Abacate Pay.
// @host			localhost:8080
// @BasePath		/
// @schemes		http
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
func main() {
	_ = godotenv.Load() // carrega .env se existir; em produção use variáveis de ambiente
	cfg := config.Load()

	database, err := db.Open(&cfg.DB)
	if err != nil {
		log.Fatalf("Erro ao conectar ao PostgreSQL: %v", err)
	}
	defer database.Close()

	mux := routes.Setup(cfg, database)

	log.Printf("Servidor iniciado em http://localhost:%s", cfg.ServerPort)
	if err := http.ListenAndServe(":"+cfg.ServerPort, mux); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}
