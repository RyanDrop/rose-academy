package models

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PostgresABPCustomerRepository implementação do ABPCustomerRepository
type PostgresABPCustomerRepository struct {
	db *sql.DB
}

// NewPostgresABPCustomerRepository cria um repositório de clientes Abacate Pay
func NewPostgresABPCustomerRepository(db *sql.DB) *PostgresABPCustomerRepository {
	return &PostgresABPCustomerRepository{db: db}
}

// Create insere um vínculo user_id <-> abp_customer_id
func (r *PostgresABPCustomerRepository) Create(c *ABPCustomer) error {
	return r.CreateInTx(nil, c)
}

// CreateInTx insere o vínculo dentro da transação tx. Se tx for nil, usa r.db.
func (r *PostgresABPCustomerRepository) CreateInTx(tx *sql.Tx, c *ABPCustomer) error {
	meta, _ := json.Marshal(c.Metadata)
	if len(meta) == 0 {
		meta = []byte("{}")
	}
	now := time.Now()
	q := `INSERT INTO abp_customers (user_id, abp_customer_id, abp_email, created_at_abp, last_sync_at, metadata, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at, updated_at`
	args := []interface{}{c.UserID, c.ABPCustomerID, c.ABPEmail, c.CreatedAtABP, c.LastSyncAt, meta, now, now}
	if tx != nil {
		return tx.QueryRow(q, args...).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	}
	return r.db.QueryRow(q, args...).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

// FindByUserID retorna o cliente ABP pelo user_id
func (r *PostgresABPCustomerRepository) FindByUserID(userID uuid.UUID) (*ABPCustomer, error) {
	row := r.db.QueryRow(`
		SELECT id, user_id, abp_customer_id, abp_email, created_at_abp, last_sync_at, metadata, created_at, updated_at
		FROM abp_customers WHERE user_id = $1
	`, userID)
	return scanABPCustomer(row)
}

// FindByABPCustomerID retorna pelo id do cliente na Abacate Pay
func (r *PostgresABPCustomerRepository) FindByABPCustomerID(abpID string) (*ABPCustomer, error) {
	row := r.db.QueryRow(`
		SELECT id, user_id, abp_customer_id, abp_email, created_at_abp, last_sync_at, metadata, created_at, updated_at
		FROM abp_customers WHERE abp_customer_id = $1
	`, abpID)
	return scanABPCustomer(row)
}

func scanABPCustomer(row *sql.Row) (*ABPCustomer, error) {
	var c ABPCustomer
	var meta []byte
	err := row.Scan(&c.ID, &c.UserID, &c.ABPCustomerID, &c.ABPEmail, &c.CreatedAtABP, &c.LastSyncAt, &meta, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(meta) > 0 {
		_ = json.Unmarshal(meta, &c.Metadata)
	}
	return &c, nil
}
