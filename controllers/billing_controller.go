package controllers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"rose-academy/api"
	"rose-academy/middleware"
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

// Subscribe cria cobrança avulsa (v1) e registro PENDING em user_subscriptions
// @Summary		Assinar plano (cobrança única)
// @Description	Cria billing na Abacate Pay v1; após pagamento o webhook marca PAID
// @Tags			billing
// @Accept		json
// @Produce		json
// @Security		BearerAuth
// @Param		body	body	SubscribeRequest	true	"produto e URLs de retorno"
// @Success		200	{object}	services.CreateSubscriptionOutput
// @Failure		400	{object}	map[string]interface{}	"validação ou cliente não vinculado"
// @Failure		401	{object}	map[string]interface{}
// @Failure		404	{object}	map[string]interface{}	"produto não encontrado"
// @Failure		503	{object}	map[string]interface{}	"gateway"
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

// CheckPixStatus consulta status do QR Code PIX
// @Summary		Status PIX
// @Tags			billing
// @Produce		json
// @Security		BearerAuth
// @Param		id	query	string	true	"id do QR code na Abacate Pay"
// @Success		200	{object}	services.CheckPixStatusOutput
// @Failure		401	{object}	map[string]interface{}
// @Failure		503	{object}	map[string]interface{}
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

// WebhookAck resposta 200 do endpoint de webhook
type WebhookAck struct {
	Received bool `json:"received"`
}

// WebhookAbacatePay recebe eventos Abacate Pay (ex.: billing.paid)
// @Summary		Webhook Abacate Pay
// @Description	Validação opcional: query webhookSecret = ABACATE_PAY_WEBHOOK_SECRET
// @Tags			webhooks
// @Accept		json
// @Produce		json
// @Param		webhookSecret	query	string	false	"deve coincidir com ABACATE_PAY_WEBHOOK_SECRET se configurado"
// @Param		body	body	AbacatePayWebhookEvent	true	"evento (ex. billing.paid)"
// @Success		200	{object}	WebhookAck
// @Failure		400	{object}	map[string]interface{}
// @Failure		401	{object}	map[string]interface{}
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
	c.billingService.HandleAbacateWebhook(event.Event, event.Data)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"received":true}`))
}
