package controllers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"rose-academy/api"
	"rose-academy/middleware"
	"rose-academy/models"
	"rose-academy/services"
)

// BillingController controller para assinaturas e webhooks Abacate Pay
type BillingController struct {
	billingService *services.BillingService
	webhookSecret  string
	devMode        bool
}

// NewBillingController cria o controller de billing
func NewBillingController(billingService *services.BillingService, webhookSecret string, devMode bool) *BillingController {
	return &BillingController{
		billingService: billingService,
		webhookSecret:  webhookSecret,
		devMode:        devMode,
	}
}

// SubscribeRequest body para POST /subscribe
type SubscribeRequest struct {
	ProductID     string `json:"product_id"`
	ReturnURL     string `json:"return_url"`
	CompletionURL string `json:"completion_url"`
}

// @Router		/subscribe [post]
func (c *BillingController) Subscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	var req SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "body JSON inválido")
		return
	}
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "product_id inválido")
		return
	}
	var fieldErrs []api.FieldError
	if strings.TrimSpace(req.ReturnURL) == "" {
		fieldErrs = append(fieldErrs, api.FieldError{Field: "return_url", Message: "é obrigatório"})
	}
	if strings.TrimSpace(req.CompletionURL) == "" {
		fieldErrs = append(fieldErrs, api.FieldError{Field: "completion_url", Message: "é obrigatório"})
	}
	if len(fieldErrs) > 0 {
		api.WriteValidation(w, r, fieldErrs)
		return
	}
	out, err := c.billingService.CreateSubscription(services.CreateSubscriptionInput{
		UserID:        userID,
		ProductID:     productID,
		ReturnURL:     strings.TrimSpace(req.ReturnURL),
		CompletionURL: strings.TrimSpace(req.CompletionURL),
	})
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			api.Write(w, r, http.StatusNotFound, "Not Found", "usuário não encontrado")
			return
		}
		if errors.Is(err, services.ErrCustomerNotLinked) {
			api.Write(w, r, http.StatusBadRequest, "Bad Request", "usuário não vinculado ao Abacate Pay. Complete o cadastro com os dados exigidos.")
			return
		}
		if errors.Is(err, services.ErrProductNotFound) {
			api.Write(w, r, http.StatusNotFound, "Not Found", "produto não encontrado")
			return
		}
		if errors.Is(err, services.ErrProductInactive) {
			api.Write(w, r, http.StatusBadRequest, "Bad Request", "produto inativo")
			return
		}
		if errors.Is(err, services.ErrAPIKeyNotConfigured) {
			api.Write(w, r, http.StatusServiceUnavailable, "Service Unavailable", "Gateway de pagamento não configurado.")
			return
		}
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(out)
}

// @Router		/payments/pix/check [get]
func (c *BillingController) CheckPixStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	_, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "parâmetro id é obrigatório")
		return
	}
	out, err := c.billingService.CheckPixStatus(id)
	if err != nil {
		if errors.Is(err, services.ErrAPIKeyNotConfigured) {
			api.Write(w, r, http.StatusServiceUnavailable, "Service Unavailable", "Gateway de pagamento não configurado.")
			return
		}
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(out)
}

// AbacatePayWebhookEvent payload do webhook (billing.paid, etc.)
type AbacatePayWebhookEvent struct {
	ID      string                 `json:"id"`
	Event   string                 `json:"event"`
	Data    map[string]interface{} `json:"data"`
	DevMode bool                   `json:"devMode"`
}

// @Router		/webhooks/abacatepay [post]
func (c *BillingController) WebhookAbacatePay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	if c.webhookSecret != "" {
		secret := r.URL.Query().Get("webhookSecret")
		if secret != c.webhookSecret {
			api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "webhook secret inválido")
			return
		}
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "corpo inválido")
		return
	}
	var event AbacatePayWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "JSON inválido")
		return
	}
	switch event.Event {
	case "billing.paid":
		billingID := extractBillingIDFromWebhook(event.Data)
		if billingID == "" {
			log.Printf("[webhook] billing.paid: não foi possível extrair billing id do payload (data=%+v)", event.Data)
		} else {
			log.Printf("[webhook] billing.paid: billing_id=%s", billingID)
			if err := c.billingService.MarkBillingPaid(billingID); err != nil {
				if errors.Is(err, models.ErrSubscriptionNotFoundForBilling) {
					log.Printf("[webhook] billing.paid: nenhuma assinatura no banco com abp_billing_id=%q (cobrança criada fora do app ou subscription não foi registrada)", billingID)
				} else {
					log.Printf("[webhook] billing.paid: erro ao atualizar banco: %v", err)
				}
			} else {
				log.Printf("[webhook] billing.paid: assinatura atualizada para PAID (billing_id=%s)", billingID)
			}
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"received":true}`))
}

func extractBillingIDFromWebhook(data map[string]interface{}) string {
	if data == nil {
		return ""
	}
	// data.billing.id (formato comum no payload da Abacate Pay)
	if b, _ := data["billing"].(map[string]interface{}); b != nil {
		if id, _ := b["id"].(string); id != "" {
			return id
		}
	}
	// data.payment.billing.id (alternativo)
	if p, _ := data["payment"].(map[string]interface{}); p != nil {
		if b, _ := p["billing"].(map[string]interface{}); b != nil {
			if id, _ := b["id"].(string); id != "" {
				return id
			}
		}
	}
	return ""
}
