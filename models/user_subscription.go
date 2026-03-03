package models

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionStatus status da assinatura/cobrança
const (
	SubscriptionStatusPending   = "PENDING"
	SubscriptionStatusPaid      = "PAID"
	SubscriptionStatusExpired    = "EXPIRED"
	SubscriptionStatusCancelled  = "CANCELLED"
	SubscriptionStatusRefunded   = "REFUNDED"
)

// UserSubscription vínculo usuário <-> cobrança Abacate Pay (tabela user_subscriptions)
type UserSubscription struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	ProductID     uuid.UUID  `json:"product_id"`
	ABPBillingID  string     `json:"abp_billing_id"`
	Status        string     `json:"status"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// UserSubscriptionRepository define operações de persistência para UserSubscription
type UserSubscriptionRepository interface {
	Create(s *UserSubscription) error
	FindByABPBillingID(abpBillingID string) (*UserSubscription, error)
	UpdateStatus(abpBillingID, status string, paidAt *time.Time) error
	FindByUserID(userID uuid.UUID) ([]*UserSubscription, error)
}
