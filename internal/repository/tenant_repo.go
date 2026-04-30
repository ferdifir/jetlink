package repository

import (
	"database/sql"
	"fmt"

	"github.com/ferdifir/jetlink/internal/models"
)

type TenantRepo struct {
	db *sql.DB
}

func NewTenantRepo(db *sql.DB) *TenantRepo {
	return &TenantRepo{db: db}
}

func (r *TenantRepo) GetBySlug(slug string) (*models.Tenant, error) {
	t := &models.Tenant{}
	query := `SELECT id, name, slug, admin_email, admin_password, fonnte_token,
		open_time, close_time, estimated_minutes, notify_before, is_active,
		COALESCE(welcome_message,''), created_at, updated_at
		FROM tenants WHERE slug = ? AND is_active = TRUE`
	err := r.db.QueryRow(query, slug).Scan(
		&t.ID, &t.Name, &t.Slug, &t.AdminEmail, &t.AdminPassword, &t.FonnteToken,
		&t.OpenTime, &t.CloseTime, &t.EstimatedMinutes, &t.NotifyBefore, &t.IsActive,
		&t.WelcomeMessage, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant by slug: %w", err)
	}
	return t, nil
}

func (r *TenantRepo) GetByID(id int) (*models.Tenant, error) {
	t := &models.Tenant{}
	query := `SELECT id, name, slug, admin_email, admin_password, fonnte_token,
		open_time, close_time, estimated_minutes, notify_before, is_active,
		COALESCE(welcome_message,''), created_at, updated_at
		FROM tenants WHERE id = ?`
	err := r.db.QueryRow(query, id).Scan(
		&t.ID, &t.Name, &t.Slug, &t.AdminEmail, &t.AdminPassword, &t.FonnteToken,
		&t.OpenTime, &t.CloseTime, &t.EstimatedMinutes, &t.NotifyBefore, &t.IsActive,
		&t.WelcomeMessage, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant by id: %w", err)
	}
	return t, nil
}

func (r *TenantRepo) List() ([]*models.Tenant, error) {
	query := `SELECT id, name, slug, admin_email, admin_password, fonnte_token,
		open_time, close_time, estimated_minutes, notify_before, is_active,
		COALESCE(welcome_message,''), created_at, updated_at
		FROM tenants ORDER BY created_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*models.Tenant
	for rows.Next() {
		t := &models.Tenant{}
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Slug, &t.AdminEmail, &t.AdminPassword, &t.FonnteToken,
			&t.OpenTime, &t.CloseTime, &t.EstimatedMinutes, &t.NotifyBefore, &t.IsActive,
			&t.WelcomeMessage, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}

func (r *TenantRepo) Create(t *models.Tenant) error {
	query := `INSERT INTO tenants (name, slug, admin_email, admin_password, fonnte_token,
		open_time, close_time, estimated_minutes, notify_before, welcome_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.Exec(query,
		t.Name, t.Slug, t.AdminEmail, t.AdminPassword, t.FonnteToken,
		t.OpenTime, t.CloseTime, t.EstimatedMinutes, t.NotifyBefore, t.WelcomeMessage,
	)
	if err != nil {
		return fmt.Errorf("create tenant: %w", err)
	}
	id, _ := result.LastInsertId()
	t.ID = int(id)
	return nil
}

func (r *TenantRepo) Update(t *models.Tenant) error {
	query := `UPDATE tenants SET name=?, slug=?, admin_email=?, fonnte_token=?,
		open_time=?, close_time=?, estimated_minutes=?, notify_before=?,
		is_active=?, welcome_message=?, updated_at=NOW()
		WHERE id=?`
	_, err := r.db.Exec(query,
		t.Name, t.Slug, t.AdminEmail, t.FonnteToken,
		t.OpenTime, t.CloseTime, t.EstimatedMinutes, t.NotifyBefore,
		t.IsActive, t.WelcomeMessage, t.ID,
	)
	if err != nil {
		return fmt.Errorf("update tenant: %w", err)
	}
	return nil
}

func (r *TenantRepo) UpdatePassword(tenantID int, hashedPassword string) error {
	_, err := r.db.Exec(`UPDATE tenants SET admin_password=?, updated_at=NOW() WHERE id=?`,
		hashedPassword, tenantID)
	return err
}

func (r *TenantRepo) SlugExists(slug string, excludeID int) (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM tenants WHERE slug=? AND id!=?`, slug, excludeID).Scan(&count)
	return count > 0, err
}

func (r *TenantRepo) Delete(id int) error {
	_, err := r.db.Exec(`UPDATE tenants SET is_active=FALSE WHERE id=?`, id)
	return err
}

func (r *TenantRepo) GetSuperadminByEmail(email string) (*models.Superadmin, error) {
	sa := &models.Superadmin{}
	err := r.db.QueryRow(`SELECT id, email, password, created_at FROM superadmins WHERE email=?`, email).
		Scan(&sa.ID, &sa.Email, &sa.Password, &sa.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sa, err
}
