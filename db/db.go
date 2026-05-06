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
	return ensureProductsAbacateSubscriptionColumn(db)
}

// ensureProductsAbacateSubscriptionColumn adiciona coluna para id do produto de assinatura na Abacate Pay (API v2).
func ensureProductsAbacateSubscriptionColumn(db *sql.DB) error {
	_, err := db.Exec(`ALTER TABLE products ADD COLUMN IF NOT EXISTS abacate_subscription_product_id VARCHAR(128)`)
	if err != nil {
		return err
	}
	if err := upsertAbacateSubscriptionProducts(db); err != nil {
		return err
	}
	if err := ensureUserSubscriptionsSubscriptionIDColumn(db); err != nil {
		return err
	}
	return ensureUserSubscriptionsCheckoutExternalIDColumn(db)
}

// ensureUserSubscriptionsSubscriptionIDColumn armazena o id da assinatura na Abacate (subs_*) para webhooks cancel/renew.
func ensureUserSubscriptionsSubscriptionIDColumn(db *sql.DB) error {
	if _, err := db.Exec(`ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS abp_subscription_id VARCHAR(128)`); err != nil {
		return err
	}
	_, err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_user_subscriptions_abp_subscription_id
		ON user_subscriptions (abp_subscription_id)
		WHERE abp_subscription_id IS NOT NULL AND abp_subscription_id <> ''
	`)
	return err
}

// ensureUserSubscriptionsCheckoutExternalIDColumn externalId enviado à Abacate (reconcilia webhooks).
func ensureUserSubscriptionsCheckoutExternalIDColumn(db *sql.DB) error {
	if _, err := db.Exec(`ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS checkout_external_id VARCHAR(128)`); err != nil {
		return err
	}
	_, err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_user_subscriptions_checkout_external_id
		ON user_subscriptions (checkout_external_id)
		WHERE checkout_external_id IS NOT NULL AND checkout_external_id <> ''
	`)
	return err
}

// upsertAbacateSubscriptionProducts garante os planos de assinatura (Abacate Pay prod_*) no catálogo local.
func upsertAbacateSubscriptionProducts(db *sql.DB) error {
	_, err := db.Exec(`
		INSERT INTO products (external_id, name, description, price_cents, is_active, abacate_subscription_product_id, updated_at)
		VALUES
			('assinatura-mensal', 'Assinatura Mensal', 'Assinatura Mensal da academia', 4999, true, 'prod_5W52efm1aj4dZt4aYSZFAdFF', now()),
			('assinatura-anual', 'Assinatura Anual', 'Assinatura Anual da academia', 49999, true, 'prod_qTrPpDnL5B3MrN5Sn3sYqqz2', now())
		ON CONFLICT (external_id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			price_cents = EXCLUDED.price_cents,
			is_active = EXCLUDED.is_active,
			abacate_subscription_product_id = EXCLUDED.abacate_subscription_product_id,
			updated_at = now()
	`)
	return err
}
