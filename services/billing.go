package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"rose-academy/models"
)

// BillingService orquestra criação de cobranças e assinaturas
type BillingService struct {
	abpClient  *AbacatePayClient
	userRepo   models.UserRepository
	abpRepo    models.ABPCustomerRepository
	productRepo models.ProductRepository
	subRepo    models.UserSubscriptionRepository
}

// NewBillingService cria o serviço de billing
func NewBillingService(
	abpClient *AbacatePayClient,
	userRepo models.UserRepository,
	abpRepo models.ABPCustomerRepository,
	productRepo models.ProductRepository,
	subRepo models.UserSubscriptionRepository,
) *BillingService {
	return &BillingService{
		abpClient:   abpClient,
		userRepo:    userRepo,
		abpRepo:     abpRepo,
		productRepo: productRepo,
		subRepo:     subRepo,
	}
}

// CreateSubscriptionInput entrada para criar assinatura/cobrança
type CreateSubscriptionInput struct {
	UserID        uuid.UUID
	ProductID     uuid.UUID
	ReturnURL     string
	CompletionURL string
}

// CreateSubscriptionOutput saída com URL de pagamento
type CreateSubscriptionOutput struct {
	BillingID   string `json:"billing_id"`
	PaymentURL  string `json:"payment_url"`
	Status      string `json:"status"`
}

var (
	ErrUserNotFound      = fmt.Errorf("usuário não encontrado")
	ErrCustomerNotLinked  = fmt.Errorf("usuário não está vinculado ao Abacate Pay; complete o cadastro")
	ErrProductNotFound   = fmt.Errorf("produto não encontrado")
	ErrProductInactive   = fmt.Errorf("produto inativo")
)

// CreateSubscription cria cobrança na Abacate Pay e registra assinatura (PENDING). Retorna URL para o usuário pagar (PIX ou cartão).
func (s *BillingService) CreateSubscription(in CreateSubscriptionInput) (*CreateSubscriptionOutput, error) {
	user, err := s.userRepo.FindByID(in.UserID)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}
	abp, err := s.abpRepo.FindByUserID(in.UserID)
	if err != nil || abp == nil {
		return nil, ErrCustomerNotLinked
	}
	product, err := s.productRepo.FindByID(in.ProductID)
	if err != nil || product == nil {
		return nil, ErrProductNotFound
	}
	if !product.IsActive {
		return nil, ErrProductInactive
	}
	billingID, paymentURL, err := s.abpClient.CreateBilling(
		abp.ABPCustomerID,
		[]BillingProduct{{
			ExternalID:  product.ExternalID,
			Name:        product.Name,
			Description: product.Description,
			Quantity:    1,
			Price:       product.PriceCents,
		}},
		in.ReturnURL,
		in.CompletionURL,
	)
	if err != nil {
		return nil, err
	}
	sub := &models.UserSubscription{
		UserID:       in.UserID,
		ProductID:    product.ID,
		ABPBillingID: billingID,
		Status:       models.SubscriptionStatusPending,
	}
	if err := s.subRepo.Create(sub); err != nil {
		return nil, err
	}
	return &CreateSubscriptionOutput{
		BillingID:  billingID,
		PaymentURL: paymentURL,
		Status:     models.SubscriptionStatusPending,
	}, nil
}

// MarkBillingPaid atualiza assinatura para PAID (chamado pelo webhook)
func (s *BillingService) MarkBillingPaid(abpBillingID string) error {
	now := time.Now()
	return s.subRepo.UpdateStatus(abpBillingID, models.SubscriptionStatusPaid, &now)
}

// CheckPixStatusOutput resultado da consulta de status do PIX
type CheckPixStatusOutput struct {
	Status    string `json:"status"`    // PENDING, EXPIRED, CANCELLED, PAID, REFUNDED
	ExpiresAt string `json:"expires_at,omitempty"`
}

// CheckPixStatus consulta o status do pagamento do QR Code PIX na Abacate Pay.
func (s *BillingService) CheckPixStatus(qrCodeID string) (*CheckPixStatusOutput, error) {
	status, expiresAt, err := s.abpClient.CheckPixQrCodeStatus(qrCodeID)
	if err != nil {
		return nil, err
	}
	return &CheckPixStatusOutput{Status: status, ExpiresAt: expiresAt}, nil
}