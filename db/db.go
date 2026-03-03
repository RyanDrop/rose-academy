package db

import (
	"database/sql"
	"log"

	"rose-academy/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

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
	if err := ensureProductsAndSubscriptionsTables(db); err != nil {
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

// ensureProductsAndSubscriptionsTables cria tabelas products e user_subscriptions se não existirem.
func ensureProductsAndSubscriptionsTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS products (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			external_id VARCHAR(128) NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			price_cents INTEGER NOT NULL CHECK (price_cents >= 100),
			image_url VARCHAR(512),
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS user_subscriptions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			product_id UUID NOT NULL REFERENCES products(id),
			abp_billing_id VARCHAR(128) NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
			paid_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE(user_id, abp_billing_id)
		);
		CREATE INDEX IF NOT EXISTS idx_user_subscriptions_user_id ON user_subscriptions(user_id);
		CREATE INDEX IF NOT EXISTS idx_user_subscriptions_abp_billing_id ON user_subscriptions(abp_billing_id);
	`)
	if err != nil {
		return err
	}
	return seedDefaultProducts(db)
}

// seedDefaultProducts insere Plano Anual e Plano Mensal se a tabela products estiver vazia.
func seedDefaultProducts(db *sql.DB) error {
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM products").Scan(&n); err != nil || n > 0 {
		return nil
	}
	_, err := db.Exec(`
		INSERT INTO products (external_id, name, description, price_cents, is_active) VALUES
		('plano-anual', 'Plano Anual', 'Plano anual do app', 49900, true),
		('plano-mensal', 'Plano Mensal', 'Plano mensal do app', 5490, true)
		ON CONFLICT (external_id) DO NOTHING
	`)
	return err
}
