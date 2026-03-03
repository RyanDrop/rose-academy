package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const (
	pathCreateCustomer   = "/customer/create"
	pathCreateBilling    = "/billing/create"
	pathPixQrCodeCheck   = "/pixQrCode/check"
)

type AbacatePayClient struct {
	baseURL        string
	apiKey         string
	client         *http.Client
	paymentMethods []string // PIX e/ou CARD; se vazio, usa só PIX
}

// NewAbacatePayClient cria o cliente. paymentMethods: ex. []string{"PIX"} ou []string{"PIX","CARD"}. Se nil/vazio, usa só PIX.
func NewAbacatePayClient(baseURL, apiKey string, paymentMethods []string) *AbacatePayClient {
	methods := paymentMethods
	if len(methods) == 0 {
		methods = []string{"PIX"}
	}
	return &AbacatePayClient{
		baseURL:        baseURL,
		apiKey:         apiKey,
		client:         &http.Client{},
		paymentMethods: methods,
	}
}

type CreateCustomerRequest struct {
	Name      string `json:"name"`
	Cellphone string `json:"cellphone"`
	Email     string `json:"email"`
	TaxID     string `json:"taxId"`
}

type CreateCustomerResponse struct {
	Data  *CustomerData `json:"data"`
	Error *string       `json:"error"`
}

type CustomerData struct {
	ID       string            `json:"id"`
	Metadata map[string]string  `json:"metadata,omitempty"`
}

// ErrAPIKeyNotConfigured é retornado quando ABACATE_PAY_API_KEY não está definida.
var ErrAPIKeyNotConfigured = fmt.Errorf("ABACATE_PAY_API_KEY não configurada")

func (c *AbacatePayClient) CreateCustomer(name, cellphone, email, taxID string) (customerID string, err error) {
	if c.apiKey == "" {
		return "", ErrAPIKeyNotConfigured
	}
	body := CreateCustomerRequest{
		Name:      name,
		Cellphone: cellphone,
		Email:     email,
		TaxID:     taxID,
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, c.baseURL+pathCreateCustomer, bytes.NewReader(raw))
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
	var out CreateCustomerResponse
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
