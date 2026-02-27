package db

import (
	"database/sql"
	"log"

	"rose-academy/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Open abre uma conexão com o PostgreSQL.
// Não cria tabelas: o esquema já existe (DDL do projeto).
// Apenas adiciona colunas cellphone e tax_id em users se não existirem (para integração Abacate Pay).
func Open(cfg *config.DBConfig) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	if err := ensureAbacatePayUserColumns(db); err != nil {
		return nil, err
	}
	log.Println("PostgreSQL conectado com sucesso")
	return db, nil
}

// ensureAbacatePayUserColumns adiciona cellphone e tax_id em users se não existirem (exigidos pela API Abacate Pay).
func ensureAbacatePayUserColumns(db *sql.DB) error {
	for _, q := range []string{
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS cellphone VARCHAR(50)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS tax_id VARCHAR(14)`,
	} {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
