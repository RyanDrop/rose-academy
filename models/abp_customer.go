package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ABPCustomer representa o vínculo usuário <-> cliente Abacate Pay (tabela abp_customers)
type ABPCustomer struct {
	ID           uuid.UUID   `json:"id"`
	UserID       uuid.UUID   `json:"user_id"`
	ABPCustomerID string     `json:"abp_customer_id"`
	ABPEmail     *string     `json:"abp_email,omitempty"`
	CreatedAtABP *time.Time  `json:"created_at_abp,omitempty"`
	LastSyncAt   time.Time   `json:"last_sync_at"`
	Metadata     ABPMetadata `json:"metadata,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// ABPMetadata metadados JSON da Abacate Pay
type ABPMetadata map[string]interface{}

func (m ABPMetadata) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

func (m *ABPMetadata) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		*m = nil
		return nil
	}
	return json.Unmarshal(b, m)
}

// ABPCustomerRepository define as operações de persistência para ABPCustomer
type ABPCustomerRepository interface {
	Create(c *ABPCustomer) error
	CreateInTx(tx *sql.Tx, c *ABPCustomer) error
	FindByUserID(userID uuid.UUID) (*ABPCustomer, error)
	FindByABPCustomerID(abpID string) (*ABPCustomer, error)
}
