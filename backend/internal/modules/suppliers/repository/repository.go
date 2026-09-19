package repository

// Suppliers repository (raw pgx, same convention as commerce/promotions/crm).
// conn() honors an ambient transaction bound via NewSupplierRepositoryWithTx.

import (
	"context"
	"strconv"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type SupplierRepository struct {
	db *database.DB
	tx *database.Tx
}

func NewSupplierRepository(db *database.DB) *SupplierRepository {
	return &SupplierRepository{db: db}
}

func NewSupplierRepositoryWithTx(tx *database.Tx) *SupplierRepository {
	return &SupplierRepository{tx: tx}
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (r *SupplierRepository) conn() querier {
	if r.tx != nil {
		return r.tx
	}
	return r.db.Pool
}

type SupplierProfile struct {
	ID          int64
	Code        string
	CompanyName string
	ContactName *string
	ContactMail *string
	ContactPhon *string
	Status      string
	UserID      *int64
	ApprovedBy  *int64
	ApprovedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SupplierContract struct {
	ID         int64
	SupplierID int64
	Version    int
	Terms      string
	ValidFrom  *time.Time
	ValidTo    *time.Time
	IsCurrent  bool
	CreatedBy  *int64
	CreatedAt  time.Time
}

const profileColumns = `id, code, company_name, contact_name, contact_email, contact_phone, status, user_id, approved_by, approved_at, created_at, updated_at`

func scanProfile(p *SupplierProfile) []any {
	return []any{&p.ID, &p.Code, &p.CompanyName, &p.ContactName, &p.ContactMail,
		&p.ContactPhon, &p.Status, &p.UserID, &p.ApprovedBy, &p.ApprovedAt,
		&p.CreatedAt, &p.UpdatedAt}
}

func (r *SupplierRepository) CreateProfile(ctx context.Context, code, company string, contactName, contactMail, contactPhone *string, userID *int64) (SupplierProfile, error) {
	var p SupplierProfile
	err := r.conn().QueryRow(ctx, `
		INSERT INTO suppliers.supplier_profile (code, company_name, contact_name, contact_email, contact_phone, status, user_id)
		VALUES ($1, $2, $3, $4, $5, 'applied', $6)
		RETURNING `+profileColumns+`
	`, code, company, contactName, contactMail, contactPhone, userID).Scan(scanProfile(&p)...)
	return p, err
}

func (r *SupplierRepository) GetProfileByID(ctx context.Context, id int64) (SupplierProfile, error) {
	var p SupplierProfile
	err := r.conn().QueryRow(ctx, `
		SELECT `+profileColumns+`
		FROM suppliers.supplier_profile WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(scanProfile(&p)...)
	return p, err
}

func (r *SupplierRepository) GetProfileByIDForUpdate(ctx context.Context, id int64) (SupplierProfile, error) {
	var p SupplierProfile
	err := r.conn().QueryRow(ctx, `
		SELECT `+profileColumns+`
		FROM suppliers.supplier_profile WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, id).Scan(scanProfile(&p)...)
	return p, err
}

func (r *SupplierRepository) GetProfileByUser(ctx context.Context, userID int64) (SupplierProfile, error) {
	var p SupplierProfile
	err := r.conn().QueryRow(ctx, `
		SELECT `+profileColumns+`
		FROM suppliers.supplier_profile WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC LIMIT 1
	`, userID).Scan(scanProfile(&p)...)
	return p, err
}

func (r *SupplierRepository) ListProfiles(ctx context.Context, status string, limit, offset int) ([]SupplierProfile, error) {
	query := `SELECT ` + profileColumns + ` FROM suppliers.supplier_profile WHERE deleted_at IS NULL`
	args := []any{}
	if status != "" {
		args = append(args, status)
		query += ` AND status = $` + strconv.Itoa(len(args)) + `::varchar`
	}
	query += ` ORDER BY created_at DESC`
	if limit > 0 {
		if limit > 200 {
			limit = 200
		}
		args = append(args, limit)
		query += ` LIMIT $` + strconv.Itoa(len(args))
	}
	if offset > 0 {
		args = append(args, offset)
		query += ` OFFSET $` + strconv.Itoa(len(args))
	}
	rows, err := r.conn().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SupplierProfile
	for rows.Next() {
		var p SupplierProfile
		if err := rows.Scan(scanProfile(&p)...); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *SupplierRepository) CountProfiles(ctx context.Context, status string) (int, error) {
	query := `SELECT COUNT(*) FROM suppliers.supplier_profile WHERE deleted_at IS NULL`
	args := []any{}
	if status != "" {
		args = append(args, status)
		query += ` AND status = $1::varchar`
	}
	var n int
	err := r.conn().QueryRow(ctx, query, args...).Scan(&n)
	return n, err
}

func (r *SupplierRepository) UpdateProfile(ctx context.Context, id int64, company, contactName, contactMail, contactPhone *string) (SupplierProfile, error) {
	setClause := `updated_at = NOW()`
	args := []any{id}
	if company != nil {
		args = append(args, *company)
		setClause += `, company_name = $` + strconv.Itoa(len(args))
	}
	if contactName != nil {
		args = append(args, *contactName)
		setClause += `, contact_name = $` + strconv.Itoa(len(args))
	}
	if contactMail != nil {
		args = append(args, *contactMail)
		setClause += `, contact_email = $` + strconv.Itoa(len(args))
	}
	if contactPhone != nil {
		args = append(args, *contactPhone)
		setClause += `, contact_phone = $` + strconv.Itoa(len(args))
	}
	var p SupplierProfile
	err := r.conn().QueryRow(ctx, `
		UPDATE suppliers.supplier_profile SET `+setClause+`
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+profileColumns+`
	`, args...).Scan(scanProfile(&p)...)
	return p, err
}

func (r *SupplierRepository) ApproveProfile(ctx context.Context, id, approver int64, userID *int64) (SupplierProfile, error) {
	var p SupplierProfile
	err := r.conn().QueryRow(ctx, `
		UPDATE suppliers.supplier_profile
		SET status = 'approved', approved_by = $2,
		    user_id = COALESCE($3, user_id),
		    approved_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL AND status = 'applied'
		RETURNING `+profileColumns+`
	`, id, approver, userID).Scan(scanProfile(&p)...)
	return p, err
}

func (r *SupplierRepository) CreateContractVersion(ctx context.Context, supplierID, version int64, terms string, validFrom, validTo *time.Time, createdBy *int64) (SupplierContract, error) {
	var c SupplierContract
	err := r.conn().QueryRow(ctx, `
		INSERT INTO suppliers.supplier_contract (supplier_id, version, terms, valid_from, valid_to, is_current, created_by)
		VALUES ($1, $2, $3, $4, $5, TRUE, $6)
		RETURNING id, supplier_id, version, terms, valid_from, valid_to, is_current, created_by, created_at
	`, supplierID, version, terms, validFrom, validTo, createdBy).Scan(
		&c.ID, &c.SupplierID, &c.Version, &c.Terms, &c.ValidFrom, &c.ValidTo, &c.IsCurrent, &c.CreatedBy, &c.CreatedAt)
	return c, err
}

func (r *SupplierRepository) DeactivateCurrentContracts(ctx context.Context, supplierID int64) error {
	_, err := r.conn().Exec(ctx, `
		UPDATE suppliers.supplier_contract SET is_current = FALSE
		WHERE supplier_id = $1 AND is_current
	`, supplierID)
	return err
}

func (r *SupplierRepository) CurrentContractVersion(ctx context.Context, supplierID int64) (int, error) {
	var v int
	err := r.conn().QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0) FROM suppliers.supplier_contract WHERE supplier_id = $1
	`, supplierID).Scan(&v)
	return v, err
}

func (r *SupplierRepository) GetContracts(ctx context.Context, supplierID int64) ([]SupplierContract, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT id, supplier_id, version, terms, valid_from, valid_to, is_current, created_by, created_at
		FROM suppliers.supplier_contract
		WHERE supplier_id = $1
		ORDER BY version DESC
	`, supplierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SupplierContract
	for rows.Next() {
		var c SupplierContract
		if err := rows.Scan(&c.ID, &c.SupplierID, &c.Version, &c.Terms, &c.ValidFrom, &c.ValidTo, &c.IsCurrent, &c.CreatedBy, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
