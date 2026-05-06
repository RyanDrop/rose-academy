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
	ID                   uuid.UUID  `json:"id"`
	UserID               uuid.UUID  `json:"user_id"`
	ProductID            uuid.UUID  `json:"product_id"`
	ABPBillingID         string     `json:"abp_billing_id"`
	ABPSubscriptionID    string     `json:"abp_subscription_id,omitempty"` // subs_* na Abacate (webhooks cancel/renew)
	CheckoutExternalID   string     `json:"checkout_external_id,omitempty"` // externalId enviado no POST /subscriptions/create
	Status               string     `json:"status"`
	PaidAt               *time.Time `json:"paid_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// UserSubscriptionRepository define operações de persistência para UserSubscription
type UserSubscriptionRepository interface {
	Create(s *UserSubscription) error
	FindByABPBillingID(abpBillingID string) (*UserSubscription, error)
	FindByABPSubscriptionID(subscriptionID string) (*UserSubscription, error)
	UpdateStatus(abpBillingID, status string, paidAt *time.Time) error
	UpdateStatusBySubscriptionID(subscriptionID, status string, paidAt *time.Time) error
	SetSubscriptionIDForBilling(billingID, subscriptionID string) error
	TouchRenewalPaidAt(billingID, subscriptionID, checkoutExt string, paidAt time.Time) error
	MarkPaidFromWebhook(billID, subID, checkoutExt string, paidAt time.Time) error
	UpdateStatusFromWebhook(billID, subID, checkoutExt, status string, paidAt *time.Time) error
	FindByUserID(userID uuid.UUID) ([]*UserSubscription, error)
	// SubscriptionAccessStatus retorna ponteiro para PAID, PENDING ou nil (JSON null) conforme prioridade PAID > PENDING; nil se não houver nenhuma dessas.
	SubscriptionAccessStatus(userID uuid.UUID) (*string, error)
}
