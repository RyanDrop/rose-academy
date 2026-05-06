package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
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
	ErrUserNotFound                         = fmt.Errorf("usuário não encontrado")
	ErrCustomerNotLinked                    = fmt.Errorf("usuário não está vinculado ao Abacate Pay; complete o cadastro")
	ErrProductNotFound                      = fmt.Errorf("produto não encontrado")
	ErrProductInactive                      = fmt.Errorf("produto inativo")
	ErrProductMissingAbacateSubscriptionID = errors.New("defina abacate_subscription_product_id no produto ou envie abp_product_id no body (produto com ciclo na Abacate Pay)")
)

// CreateSubscription cria cobrança na Abacate Pay e registra assinatura (PENDING).
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

// SubscriptionAccessStatus delega ao repositório (PAID, PENDING ou nil).
func (s *BillingService) SubscriptionAccessStatus(userID uuid.UUID) (*string, error) {
	return s.subRepo.SubscriptionAccessStatus(userID)
}

// CreateAbacateSubscriptionCheckoutInput checkout de assinatura (API v2 — produto com ciclo na loja).
type CreateAbacateSubscriptionCheckoutInput struct {
	UserID        uuid.UUID
	ProductID     uuid.UUID
	AbpProductID  string // opcional: sobrescreve abacate_subscription_product_id do produto
	ReturnURL     string
	CompletionURL string
	Methods       []string // opcional; padrão ABACATE_PAY_SUBSCRIPTION_METHODS
}

// CreateAbacateSubscriptionCheckoutOutput mesmo padrão do subscribe one-shot, com dados extras da v2.
type CreateAbacateSubscriptionCheckoutOutput struct {
	// UserSubscriptionID registro local em user_subscriptions (status PENDING até o webhook marcar pago).
	UserSubscriptionID uuid.UUID `json:"user_subscription_id"`
	BillingID          string    `json:"billing_id"`
	PaymentURL         string    `json:"payment_url"`
	Status             string    `json:"status"`
	AmountCents        int       `json:"amount_cents,omitempty"`
	// DevMode true quando a Abacate Pay cria o checkout em ambiente de testes (chave de API de dev/sandbox).
	DevMode bool `json:"dev_mode"`
	// Hint mensagem curta quando DevMode for relevante para achar o checkout no painel.
	Hint string `json:"hint,omitempty"`
}

// CreateAbacateSubscriptionCheckout cria checkout de assinatura recorrente na Abacate Pay (POST /v2/subscriptions/create).
func (s *BillingService) CreateAbacateSubscriptionCheckout(in CreateAbacateSubscriptionCheckoutInput) (*CreateAbacateSubscriptionCheckoutOutput, error) {
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
	abpProdID := in.AbpProductID
	if abpProdID == "" {
		abpProdID = product.AbacateSubscriptionProductID
	}
	if abpProdID == "" {
		return nil, ErrProductMissingAbacateSubscriptionID
	}
	req := CreateSubscriptionCheckoutRequest{
		Items: []SubscriptionCheckoutItem{{
			ID:       abpProdID,
			Quantity: 1,
		}},
		CustomerID:    abp.ABPCustomerID,
		ReturnURL:     in.ReturnURL,
		CompletionURL: in.CompletionURL,
		Methods:       in.Methods,
	}
	data, err := s.abpClient.CreateSubscriptionCheckout(req)
	if err != nil {
		return nil, err
	}
	sub := &models.UserSubscription{
		UserID:       in.UserID,
		ProductID:    product.ID,
		ABPBillingID: data.ID,
		Status:       models.SubscriptionStatusPending,
	}
	if err := s.subRepo.Create(sub); err != nil {
		return nil, err
	}
	log.Printf("[billing] user_subscriptions PENDING id=%s billing_id=%s user_id=%s devMode=%v payment_url=%s",
		sub.ID, data.ID, in.UserID, data.DevMode, data.URL)
	amountCents := int(data.Amount)
	out := &CreateAbacateSubscriptionCheckoutOutput{
		UserSubscriptionID: sub.ID,
		BillingID:          data.ID,
		PaymentURL:         data.URL,
		Status:             data.Status,
		AmountCents:        amountCents,
		DevMode:            data.DevMode,
	}
	if data.DevMode {
		out.Hint = "Checkout em modo de testes (devMode). No painel Abacate Pay, abra a área de sandbox / ambiente de desenvolvimento ou o filtro de cobranças de teste — ele não aparece só na visão de produção."
	}
	return out, nil
}

// userSubscriptionListItem payload em GET /abacate/subscriptions (fonte: user_subscriptions).
type userSubscriptionListItem struct {
	ID                 string  `json:"id"` // subs_* se houver; senão bill_*
	UserSubscriptionID string  `json:"user_subscription_id"`
	ProductID          string  `json:"product_id"`
	BillingID          string  `json:"billing_id"`
	SubscriptionID     string  `json:"subscription_id,omitempty"`
	CheckoutExternalID string  `json:"checkout_external_id,omitempty"`
	Status             string  `json:"status"`
	PaidAt             *string `json:"paid_at,omitempty"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

// ListAbacateSubscriptions lista assinaturas só em user_subscriptions (filtro user_id). Não chama API Abacate. Mantém no máximo uma linha PAID (a mais recente por paid_at, depois created_at).
func (s *BillingService) ListAbacateSubscriptions(userID uuid.UUID, p ListSubscriptionsParams) (json.RawMessage, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	subs, err := s.subRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	subs = dedupePaidUserSubscriptions(subs)
	subs = filterUserSubscriptionsByQuery(subs, p)
	items := make([]userSubscriptionListItem, 0, len(subs))
	for _, s := range subs {
		items = append(items, toUserSubscriptionListItem(s))
	}
	out := struct {
		Success bool                       `json:"success"`
		Data    []userSubscriptionListItem `json:"data"`
		Error   interface{}                `json:"error"`
	}{Success: true, Data: items, Error: nil}
	return json.Marshal(out)
}

func paidAtOrCreated(s *models.UserSubscription) time.Time {
	if s.PaidAt != nil {
		return *s.PaidAt
	}
	return s.CreatedAt
}

// dedupePaidUserSubscriptions mantém no máximo um registro PAID (o mais “recente”); demais status permanecem.
func dedupePaidUserSubscriptions(subs []*models.UserSubscription) []*models.UserSubscription {
	var paid []*models.UserSubscription
	var other []*models.UserSubscription
	for _, s := range subs {
		if s.Status == models.SubscriptionStatusPaid {
			paid = append(paid, s)
		} else {
			other = append(other, s)
		}
	}
	if len(paid) <= 1 {
		return subs
	}
	sort.Slice(paid, func(i, j int) bool {
		return paidAtOrCreated(paid[i]).After(paidAtOrCreated(paid[j]))
	})
	merged := make([]*models.UserSubscription, 0, len(other)+1)
	merged = append(merged, paid[0])
	merged = append(merged, other...)
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].CreatedAt.After(merged[j].CreatedAt)
	})
	return merged
}

func filterUserSubscriptionsByQuery(subs []*models.UserSubscription, p ListSubscriptionsParams) []*models.UserSubscription {
	out := subs
	if id := strings.TrimSpace(p.ID); id != "" {
		var next []*models.UserSubscription
		for _, s := range out {
			if s.ABPSubscriptionID == id || s.ABPBillingID == id || s.ID.String() == id || strings.TrimSpace(s.CheckoutExternalID) == id {
				next = append(next, s)
			}
		}
		out = next
	}
	if ext := strings.TrimSpace(p.ExternalID); ext != "" {
		var next []*models.UserSubscription
		for _, s := range out {
			if strings.TrimSpace(s.CheckoutExternalID) == ext {
				next = append(next, s)
			}
		}
		out = next
	}
	if st := strings.TrimSpace(p.Status); st != "" {
		want := strings.ToUpper(st)
		var next []*models.UserSubscription
		for _, s := range out {
			if strings.ToUpper(strings.TrimSpace(s.Status)) == want {
				next = append(next, s)
			}
		}
		out = next
	}
	if p.Limit > 0 && len(out) > p.Limit {
		out = out[:p.Limit]
	}
	return out
}

func toUserSubscriptionListItem(s *models.UserSubscription) userSubscriptionListItem {
	id := strings.TrimSpace(s.ABPSubscriptionID)
	if id == "" {
		id = s.ABPBillingID
	}
	var paidAt *string
	if s.PaidAt != nil {
		t := s.PaidAt.UTC().Format(time.RFC3339Nano)
		paidAt = &t
	}
	return userSubscriptionListItem{
		ID:                 id,
		UserSubscriptionID: s.ID.String(),
		ProductID:          s.ProductID.String(),
		BillingID:          s.ABPBillingID,
		SubscriptionID:     strings.TrimSpace(s.ABPSubscriptionID),
		CheckoutExternalID: strings.TrimSpace(s.CheckoutExternalID),
		Status:             s.Status,
		PaidAt:             paidAt,
		CreatedAt:          s.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:          s.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

// CancelAbacateSubscription cancela assinatura na Abacate Pay (id ex.: subs_...).
func (s *BillingService) CancelAbacateSubscription(subscriptionID string) (json.RawMessage, error) {
	return s.abpClient.CancelSubscription(subscriptionID)
}