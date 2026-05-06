package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	// Clientes: API v2 (chaves atuais do dashboard são v2; v1 retorna "API key version mismatch").
	pathCustomersCreateV2     = "/customers/create"
	pathCreateBilling         = "/billing/create"
	pathPixQrCodeCheck        = "/pixQrCode/check"
	pathSubscriptionsCreate   = "/subscriptions/create"
	pathSubscriptionsList     = "/subscriptions/list"
	pathSubscriptionsCancel   = "/subscriptions/cancel"
)

type AbacatePayClient struct {
	baseURL             string
	baseURLV2           string
	apiKey              string
	client              *http.Client
	paymentMethods      []string // v1 billing: PIX e/ou CARD
	subscriptionMethods []string // v2 subscriptions/create; padrão CARD
}

// NewAbacatePayClient cria o cliente. baseURLV2: API v2 (assinaturas), ex. https://api.abacatepay.com/v2. Se vazio, usa v2 padrão.
func NewAbacatePayClient(baseURL, baseURLV2, apiKey string, paymentMethods, subscriptionMethods []string) *AbacatePayClient {
	methods := paymentMethods
	if len(methods) == 0 {
		methods = []string{"PIX"}
	}
	subM := subscriptionMethods
	if len(subM) == 0 {
		subM = []string{"CARD"}
	}
	v2 := baseURLV2
	if v2 == "" {
		v2 = "https://api.abacatepay.com/v2"
	}
	return &AbacatePayClient{
		baseURL:             baseURL,
		baseURLV2:           strings.TrimRight(v2, "/"),
		apiKey:              apiKey,
		client:              &http.Client{},
		paymentMethods:      methods,
		subscriptionMethods: subM,
	}
}

// Body plano conforme POST /v2/customers/create (o servidor valida email na raiz, não em data.*).
type createCustomerV2Body struct {
	Email     string `json:"email"`
	Name      string `json:"name,omitempty"`
	Cellphone string `json:"cellphone,omitempty"`
	TaxID     string `json:"taxId,omitempty"`
}

type createCustomerV2Response struct {
	Data  *customerCreatedData `json:"data"`
	Error *string              `json:"error"`
}

type customerCreatedData struct {
	ID string `json:"id"`
}

// ErrAPIKeyNotConfigured é retornado quando ABACATE_PAY_API_KEY não está definida.
var ErrAPIKeyNotConfigured = fmt.Errorf("ABACATE_PAY_API_KEY não configurada")

// CreateCustomer cria cliente na Abacate Pay (POST /v2/customers/create). Body JSON plano; apenas email é obrigatório.
func (c *AbacatePayClient) CreateCustomer(name, cellphone, email, taxID string) (customerID string, err error) {
	if c.apiKey == "" {
		return "", ErrAPIKeyNotConfigured
	}
	body := createCustomerV2Body{Email: email}
	if name != "" {
		body.Name = name
	}
	if cellphone != "" {
		body.Cellphone = cellphone
	}
	if taxID != "" {
		body.TaxID = taxID
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	url := c.baseURLV2 + pathCustomersCreateV2
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out createCustomerV2Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Error != nil {
		return "", fmt.Errorf("abacate pay: %s", *out.Error)
	}
	if out.Data == nil || out.Data.ID == "" {
		return "", fmt.Errorf("abacate pay: resposta sem id do cliente")
	}
	return out.Data.ID, nil
}

// BillingProduct produto enviado na criação da cobrança (API Abacate Pay)
type BillingProduct struct {
	ExternalID  string `json:"externalId"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Quantity    int    `json:"quantity"`
	Price       int    `json:"price"` // centavos
}

// CreateBillingRequest body para POST /billing/create
type CreateBillingRequest struct {
	Frequency     string         `json:"frequency"`     // ONE_TIME ou MULTIPLE_PAYMENTS
	Methods       []string       `json:"methods"`       // ["PIX", "CARD"]
	Products      []BillingProduct `json:"products"`
	ReturnURL     string         `json:"returnUrl"`
	CompletionURL string         `json:"completionUrl"`
	CustomerID    string         `json:"customerId,omitempty"`
	AllowCoupons  bool           `json:"allowCoupons,omitempty"`
}

// CreateBillingResponse resposta da criação de cobrança
type CreateBillingResponse struct {
	Data  *BillingData `json:"data"`
	Error *string     `json:"error"`
}

// BillingData dados da cobrança retornados pela API
type BillingData struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Status   string `json:"status"`
	DevMode  bool   `json:"devMode"`
	Methods  []string `json:"methods"`
	Products []struct {
		ID         string `json:"id"`
		ExternalID string `json:"externalId"`
		Quantity   int    `json:"quantity"`
	} `json:"products"`
	Frequency string `json:"frequency"`
}

// CreateBilling cria uma cobrança na Abacate Pay (PIX e cartão). customerID deve ser o id do cliente já cadastrado.
func (c *AbacatePayClient) CreateBilling(customerID string, products []BillingProduct, returnURL, completionURL string) (billingID, paymentURL string, err error) {
	if c.apiKey == "" {
		return "", "", ErrAPIKeyNotConfigured
	}
	if len(products) == 0 {
		return "", "", fmt.Errorf("abacate pay: é necessário ao menos um produto")
	}
	body := CreateBillingRequest{
		Frequency:     "ONE_TIME",
		Methods:       c.paymentMethods,
		Products:      products,
		ReturnURL:     returnURL,
		CompletionURL: completionURL,
		CustomerID:    customerID,
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, c.baseURL+pathCreateBilling, bytes.NewReader(raw))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	var out CreateBillingResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", err
	}
	if out.Error != nil {
		return "", "", fmt.Errorf("abacate pay: %s", *out.Error)
	}
	if out.Data == nil || out.Data.ID == "" {
		return "", "", fmt.Errorf("abacate pay: resposta sem id da cobrança")
	}
	if out.Data.URL == "" {
		return "", "", fmt.Errorf("abacate pay: resposta sem url de pagamento")
	}
	return out.Data.ID, out.Data.URL, nil
}

// PixQrCodeCheckResponse resposta de GET /pixQrCode/check (docs.abacatepay.com/pages/pix-qrcode/check)
type PixQrCodeCheckResponse struct {
	Data  *PixQrCodeCheckData `json:"data"`
	Error *string             `json:"error"`
}

// PixQrCodeCheckData status do QR Code PIX
type PixQrCodeCheckData struct {
	Status    string `json:"status"`    // PENDING, EXPIRED, CANCELLED, PAID, REFUNDED
	ExpiresAt string `json:"expiresAt"` // ISO 8601
}

// CheckPixQrCodeStatus consulta o status do pagamento do QR Code PIX na Abacate Pay.
func (c *AbacatePayClient) CheckPixQrCodeStatus(qrCodeID string) (status, expiresAt string, err error) {
	if c.apiKey == "" {
		return "", "", ErrAPIKeyNotConfigured
	}
	if qrCodeID == "" {
		return "", "", fmt.Errorf("id do QR Code PIX é obrigatório")
	}
	u, _ := url.Parse(c.baseURL + pathPixQrCodeCheck)
	q := u.Query()
	q.Set("id", qrCodeID)
	u.RawQuery = q.Encode()
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	var out PixQrCodeCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", err
	}
	if out.Error != nil {
		return "", "", fmt.Errorf("abacate pay: %s", *out.Error)
	}
	if out.Data == nil {
		return "", "", fmt.Errorf("abacate pay: resposta sem dados")
	}
	return out.Data.Status, out.Data.ExpiresAt, nil
}

// --- API v2: assinaturas (https://docs.abacatepay.com/pages/subscriptions/reference) ---

// SubscriptionCheckoutItem item único do checkout de assinatura (exatamente 1)
type SubscriptionCheckoutItem struct {
	ID       string `json:"id"`
	Quantity int    `json:"quantity"`
}

// CreateSubscriptionCheckoutRequest body POST /v2/subscriptions/create
type CreateSubscriptionCheckoutRequest struct {
	Items           []SubscriptionCheckoutItem `json:"items"`
	CustomerID      string                       `json:"customerId,omitempty"`
	ExternalID      string                       `json:"externalId,omitempty"`
	ReturnURL       string                       `json:"returnUrl,omitempty"`
	CompletionURL   string                       `json:"completionUrl,omitempty"`
	Methods         []string                     `json:"methods,omitempty"`
	Metadata        map[string]interface{}       `json:"metadata,omitempty"`
	Coupons         []string                     `json:"coupons,omitempty"`
}

// AbacateSubscriptionCheckoutData resposta data do checkout de assinatura
type AbacateSubscriptionCheckoutData struct {
	ID         string `json:"id"`
	ExternalID string `json:"externalId"`
	// Algumas respostas expõem o id só em billing.id
	Billing *struct {
		ID string `json:"id"`
	} `json:"billing"`
	URL           string  `json:"url"`
	Status        string  `json:"status"`
	Amount        float64 `json:"amount"`
	CustomerID    string  `json:"customerId"`
	ReturnURL     string  `json:"returnUrl"`
	CompletionURL string  `json:"completionUrl"`
	DevMode       bool    `json:"devMode"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

type abacateV2Envelope struct {
	Data    json.RawMessage `json:"data"`
	Success *bool           `json:"success"`
	Error   *string         `json:"error"`
}

func (c *AbacatePayClient) parseV2Envelope(body []byte, into interface{}) error {
	var env abacateV2Envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return err
	}
	if env.Error != nil && *env.Error != "" {
		return fmt.Errorf("abacate pay: %s", *env.Error)
	}
	if env.Success != nil && !*env.Success {
		msg := "requisição não bem-sucedida"
		if env.Error != nil {
			msg = *env.Error
		}
		return fmt.Errorf("abacate pay: %s", msg)
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return fmt.Errorf("abacate pay: resposta sem data")
	}
	return json.Unmarshal(env.Data, into)
}

// subscriptionCheckoutMethods a API POST /subscriptions/create só aceita CARD; incluir PIX (ex. ["CARD","PIX"])
// provoca 422 "Expected kind 'UnionEnum'" no validador da Abacate Pay.
func subscriptionCheckoutMethods(requested []string, fallback []string) []string {
	for _, m := range requested {
		if strings.EqualFold(strings.TrimSpace(m), "CARD") {
			return []string{"CARD"}
		}
	}
	for _, m := range fallback {
		if strings.EqualFold(strings.TrimSpace(m), "CARD") {
			return []string{"CARD"}
		}
	}
	return []string{"CARD"}
}

// CreateSubscriptionCheckout cria checkout de assinatura (produto com ciclo na loja Abacate Pay).
func (c *AbacatePayClient) CreateSubscriptionCheckout(req CreateSubscriptionCheckoutRequest) (*AbacateSubscriptionCheckoutData, error) {
	if c.apiKey == "" {
		return nil, ErrAPIKeyNotConfigured
	}
	if len(req.Items) != 1 {
		return nil, fmt.Errorf("abacate pay: assinatura exige exatamente um item")
	}
	methods := req.Methods
	if len(methods) == 0 {
		methods = c.subscriptionMethods
	}
	req.Methods = subscriptionCheckoutMethods(methods, c.subscriptionMethods)
	raw, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, c.baseURLV2+pathSubscriptionsCreate, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("abacate pay: HTTP %d: %s", resp.StatusCode, string(body))
	}
	var data AbacateSubscriptionCheckoutData
	if err := c.parseV2Envelope(body, &data); err != nil {
		return nil, err
	}
	if data.ID == "" && data.Billing != nil {
		data.ID = strings.TrimSpace(data.Billing.ID)
	}
	if data.ID == "" || data.URL == "" {
		return nil, fmt.Errorf("abacate pay: resposta de assinatura sem id ou url")
	}
	return &data, nil
}

// ListSubscriptionsParams query GET /v2/subscriptions/list
type ListSubscriptionsParams struct {
	Limit      int
	After      string
	Before     string
	ID         string
	ExternalID string
	Status     string
	Email      string
	TaxID      string
}

// ListSubscriptions lista checkouts de assinatura (JSON bruto da API v2).
func (c *AbacatePayClient) ListSubscriptions(p ListSubscriptionsParams) (json.RawMessage, error) {
	if c.apiKey == "" {
		return nil, ErrAPIKeyNotConfigured
	}
	q := url.Values{}
	if p.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", p.Limit))
	}
	if p.After != "" {
		q.Set("after", p.After)
	}
	if p.Before != "" {
		q.Set("before", p.Before)
	}
	if p.ID != "" {
		q.Set("id", p.ID)
	}
	if p.ExternalID != "" {
		q.Set("externalId", p.ExternalID)
	}
	if p.Status != "" {
		q.Set("status", p.Status)
	}
	if p.Email != "" {
		q.Set("email", p.Email)
	}
	if p.TaxID != "" {
		q.Set("taxId", p.TaxID)
	}
	u := c.baseURLV2 + pathSubscriptionsList
	if enc := q.Encode(); enc != "" {
		u += "?" + enc
	}
	httpReq, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("abacate pay: HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.RawMessage(body), nil
}

// CancelSubscriptionRequest body POST /v2/subscriptions/cancel
type CancelSubscriptionRequest struct {
	ID string `json:"id"`
}

// CancelSubscription cancela assinatura ativa (id retornado pela API Abacate Pay, ex. subs_...).
func (c *AbacatePayClient) CancelSubscription(subscriptionID string) (json.RawMessage, error) {
	if c.apiKey == "" {
		return nil, ErrAPIKeyNotConfigured
	}
	if subscriptionID == "" {
		return nil, fmt.Errorf("id da assinatura é obrigatório")
	}
	raw, _ := json.Marshal(CancelSubscriptionRequest{ID: subscriptionID})
	httpReq, err := http.NewRequest(http.MethodPost, c.baseURLV2+pathSubscriptionsCancel, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("abacate pay: HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.RawMessage(body), nil
}
