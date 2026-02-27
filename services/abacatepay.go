package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	pathCreateCustomer = "/customer/create"
)

type AbacatePayClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewAbacatePayClient(baseURL, apiKey string) *AbacatePayClient {
	return &AbacatePayClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{},
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
