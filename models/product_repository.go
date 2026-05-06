package models

import (
	"database/sql"

	"github.com/google/uuid"
)

// PostgresProductRepository implementação do ProductRepository
type PostgresProductRepository struct {
	db *sql.DB
}

// NewPostgresProductRepository cria repositório de produtos
func NewPostgresProductRepository(db *sql.DB) *PostgresProductRepository {
	return &PostgresProductRepository{db: db}
}

// FindAll retorna todos os produtos ativos
func (r *PostgresProductRepository) FindAll() ([]*Product, error) {
	rows, err := r.db.Query(`
		SELECT id, external_id, name, COALESCE(description,''), price_cents, COALESCE(image_url,''), COALESCE(abacate_subscription_product_id,''), is_active, created_at, updated_at
		FROM products WHERE is_active = true ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Product
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.ID, &p.ExternalID, &p.Name, &p.Description, &p.PriceCents, &p.ImageURL, &p.AbacateSubscriptionProductID, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, &p)
	}
	return list, rows.Err()
}

// FindByID retorna produto pelo ID
func (r *PostgresProductRepository) FindByID(id uuid.UUID) (*Product, error) {
	row := r.db.QueryRow(`
		SELECT id, external_id, name, COALESCE(description,''), price_cents, COALESCE(image_url,''), COALESCE(abacate_subscription_product_id,''), is_active, created_at, updated_at
		FROM products WHERE id = $1
	`, id)
	var p Product
	err := row.Scan(&p.ID, &p.ExternalID, &p.Name, &p.Description, &p.PriceCents, &p.ImageURL, &p.AbacateSubscriptionProductID, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// FindByExternalID retorna produto pelo external_id
func (r *PostgresProductRepository) FindByExternalID(externalID string) (*Product, error) {
	row := r.db.QueryRow(`
		SELECT id, external_id, name, COALESCE(description,''), price_cents, COALESCE(image_url,''), COALESCE(abacate_subscription_product_id,''), is_active, created_at, updated_at
		FROM products WHERE external_id = $1
	`, externalID)
	var p Product
	err := row.Scan(&p.ID, &p.ExternalID, &p.Name, &p.Description, &p.PriceCents, &p.ImageURL, &p.AbacateSubscriptionProductID, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
