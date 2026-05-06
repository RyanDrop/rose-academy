package models

import (
	"time"

	"github.com/google/uuid"
)

// Product representa um plano/produto vendido (tabela products)
type Product struct {
	ID                           uuid.UUID `json:"id"`
	ExternalID                   string    `json:"external_id"`
	Name                         string    `json:"name"`
	Description                  string    `json:"description"`
	PriceCents                   int       `json:"price_cents"`
	ImageURL                     string    `json:"image_url,omitempty"`
	AbacateSubscriptionProductID string    `json:"abacate_subscription_product_id,omitempty"` // id do produto com ciclo na loja Abacate Pay (checkout v2)
	IsActive                     bool      `json:"is_active"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`
}

// ProductRepository define operações de persistência para Product
type ProductRepository interface {
	FindAll() ([]*Product, error)
	FindByID(id uuid.UUID) (*Product, error)
	FindByExternalID(externalID string) (*Product, error)
}
