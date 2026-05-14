// Aplica db/seed_weekly_workouts.sql no PostgreSQL (usa .env como a API).
// Se o seed falhar ao atualizar exercicios por FK, rode antes no psql:
//
//	psql "$DATABASE_URL" -f db/wipe_workouts_and_training_progress.sql
//
// Na raiz do repositório:
//
//	go run ./cmd/seedworkouts
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"

	"rose-academy/config"
	"rose-academy/db"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	database, err := db.Open(&cfg.DB)
	if err != nil {
		log.Fatalf("banco: %v", err)
	}
	defer database.Close()

	sqlPath := filepath.Join("db", "seed_weekly_workouts.sql")
	if len(os.Args) > 1 {
		sqlPath = os.Args[1]
	}
	b, err := os.ReadFile(sqlPath)
	if err != nil {
		log.Fatalf("ler %s: %v", sqlPath, err)
	}
	if _, err := database.Exec(string(b)); err != nil {
		log.Fatalf("seed SQL: %v", err)
	}
	log.Printf("OK: %s aplicado.", sqlPath)
}
