package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ferdifir/jetlink/internal/models"
)

type CustomerRepo struct {
	db *sql.DB
}

func NewCustomerRepo(db *sql.DB) *CustomerRepo {
	return &CustomerRepo{db: db}
}

func (r *CustomerRepo) GetByPhone(tenantID int, phone string) (*models.Customer, error) {
	c := &models.Customer{}
	err := r.db.QueryRow(
		`SELECT id, tenant_id, phone, name, created_at FROM customers WHERE tenant_id=? AND phone=?`,
		tenantID, phone,
	).Scan(&c.ID, &c.TenantID, &c.Phone, &c.Name, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get customer by phone: %w", err)
	}
	return c, nil
}

func (r *CustomerRepo) GetByID(id int) (*models.Customer, error) {
	c := &models.Customer{}
	err := r.db.QueryRow(
		`SELECT id, tenant_id, phone, name, created_at FROM customers WHERE id=?`, id,
	).Scan(&c.ID, &c.TenantID, &c.Phone, &c.Name, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get customer by id: %w", err)
	}
	return c, nil
}

func (r *CustomerRepo) Create(c *models.Customer) error {
	result, err := r.db.Exec(
		`INSERT INTO customers (tenant_id, phone, name) VALUES (?, ?, ?)`,
		c.TenantID, c.Phone, c.Name,
	)
	if err != nil {
		return fmt.Errorf("create customer: %w", err)
	}
	id, _ := result.LastInsertId()
	c.ID = int(id)
	return nil
}

func (r *CustomerRepo) SaveOTP(tenantID int, phone, code string) error {
	expiresAt := time.Now().Add(5 * time.Minute)
	_, err := r.db.Exec(
		`INSERT INTO otp_codes (tenant_id, phone, code, expires_at) VALUES (?, ?, ?, ?)`,
		tenantID, phone, code, expiresAt,
	)
	return err
}

func (r *CustomerRepo) VerifyOTP(tenantID int, phone, code string) (bool, error) {
	var id int
	err := r.db.QueryRow(
		`SELECT id FROM otp_codes WHERE tenant_id=? AND phone=? AND code=? AND used=FALSE AND expires_at > NOW()
		 ORDER BY created_at DESC LIMIT 1`,
		tenantID, phone, code,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	_, err = r.db.Exec(`UPDATE otp_codes SET used=TRUE WHERE id=?`, id)
	return true, err
}
