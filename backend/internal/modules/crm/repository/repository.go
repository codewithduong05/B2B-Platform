package repository

// CRM repository (raw pgx, same convention as commerce/promotions).
// conn() honors an ambient transaction bound via NewCRMRepositoryWithTx.

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type CRMRepository struct {
	db *database.DB
	tx *database.Tx
}

func NewCRMRepository(db *database.DB) *CRMRepository {
	return &CRMRepository{db: db}
}

func NewCRMRepositoryWithTx(tx *database.Tx) *CRMRepository {
	return &CRMRepository{tx: tx}
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (r *CRMRepository) conn() querier {
	if r.tx != nil {
		return r.tx
	}
	return r.db.Pool
}

type Partner struct {
	ID        int64
	Code      string
	Name      string
	Slug      string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ReferralCode struct {
	ID        int64
	Code      string
	PartnerID int64
	IsActive  bool
	CreatedAt time.Time
}

type Lead struct {
	ID          int64
	Code        string
	ContactName string
	Business    string
	Email       *string
	Phone       *string
	Message     *string
	Status      string
	PartnerID   *int64
	ReferralID  *int64
	AssignedTo  *int64
	BuyerID     *int64
	Notes       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const partnerColumns = `id, code, name, slug, is_active, created_at, updated_at`

func (r *CRMRepository) CreatePartner(ctx context.Context, code, name, slug string) (Partner, error) {
	var p Partner
	err := r.conn().QueryRow(ctx, `
		INSERT INTO crm.partner (code, name, slug)
		VALUES ($1, $2, $3)
		RETURNING `+partnerColumns+`
	`, code, name, slug).Scan(&p.ID, &p.Code, &p.Name, &p.Slug, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (r *CRMRepository) GetPartnerByID(ctx context.Context, id int64) (Partner, error) {
	var p Partner
	err := r.conn().QueryRow(ctx, `
		SELECT `+partnerColumns+`
		FROM crm.partner WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&p.ID, &p.Code, &p.Name, &p.Slug, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (r *CRMRepository) ResolvePartnerRef(ctx context.Context, ref string) (Partner, *ReferralCode, error) {
	// Referral code first, then partner slug. Both must be active.
	var rc ReferralCode
	if err := r.conn().QueryRow(ctx, `
		SELECT r.id, r.code, r.partner_id, r.is_active, r.created_at
		FROM crm.referral_code r
		WHERE r.code = $1 AND r.deleted_at IS NULL AND r.is_active
	`, ref).Scan(&rc.ID, &rc.Code, &rc.PartnerID, &rc.IsActive, &rc.CreatedAt); err == nil {
		p, perr := r.GetPartnerByID(ctx, rc.PartnerID)
		if perr == nil && p.IsActive {
			return p, &rc, nil
		}
		if perr != nil && !errors.Is(perr, pgx.ErrNoRows) {
			return Partner{}, nil, perr
		}
	}
	var p Partner
	if err := r.conn().QueryRow(ctx, `
		SELECT id, code, name, slug, is_active, created_at, updated_at
		FROM crm.partner
		WHERE slug = $1 AND deleted_at IS NULL AND is_active
	`, ref).Scan(&p.ID, &p.Code, &p.Name, &p.Slug, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return Partner{}, nil, err
	}
	return p, nil, nil
}

func (r *CRMRepository) ListPartners(ctx context.Context, limit, offset int) ([]Partner, error) {
	query := `SELECT ` + partnerColumns + ` FROM crm.partner WHERE deleted_at IS NULL ORDER BY created_at DESC`
	args := []any{}
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

	var out []Partner
	for rows.Next() {
		var p Partner
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.Slug, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *CRMRepository) CountPartners(ctx context.Context) (int, error) {
	var n int
	err := r.conn().QueryRow(ctx, `SELECT COUNT(*) FROM crm.partner WHERE deleted_at IS NULL`).Scan(&n)
	return n, err
}

func (r *CRMRepository) CreateReferral(ctx context.Context, code string, partnerID int64) (ReferralCode, error) {
	var rc ReferralCode
	err := r.conn().QueryRow(ctx, `
		INSERT INTO crm.referral_code (code, partner_id)
		VALUES ($1, $2)
		RETURNING id, code, partner_id, is_active, created_at
	`, code, partnerID).Scan(&rc.ID, &rc.Code, &rc.PartnerID, &rc.IsActive, &rc.CreatedAt)
	return rc, err
}

func (r *CRMRepository) ListReferrals(ctx context.Context, limit, offset int) ([]ReferralCode, error) {
	query := `SELECT id, code, partner_id, is_active, created_at FROM crm.referral_code WHERE deleted_at IS NULL ORDER BY created_at DESC`
	args := []any{}
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

	var out []ReferralCode
	for rows.Next() {
		var rc ReferralCode
		if err := rows.Scan(&rc.ID, &rc.Code, &rc.PartnerID, &rc.IsActive, &rc.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, rc)
	}
	return out, rows.Err()
}

func (r *CRMRepository) CountReferrals(ctx context.Context) (int, error) {
	var n int
	err := r.conn().QueryRow(ctx, `SELECT COUNT(*) FROM crm.referral_code WHERE deleted_at IS NULL`).Scan(&n)
	return n, err
}

func (r *CRMRepository) ReferralUses(ctx context.Context, referralID int64) (int, error) {
	var n int
	err := r.conn().QueryRow(ctx, `SELECT COUNT(*) FROM crm.lead WHERE referral_id = $1 AND deleted_at IS NULL`, referralID).Scan(&n)
	return n, err
}

const leadColumns = `id, code, contact_name, business_name, email, phone, message, status, partner_id, referral_id, assigned_to, buyer_id, notes, created_at, updated_at`

func scanLead(l *Lead) []any {
	return []any{&l.ID, &l.Code, &l.ContactName, &l.Business, &l.Email, &l.Phone,
		&l.Message, &l.Status, &l.PartnerID, &l.ReferralID, &l.AssignedTo,
		&l.BuyerID, &l.Notes, &l.CreatedAt, &l.UpdatedAt}
}

func (r *CRMRepository) CreateLead(ctx context.Context, code, contactName, business string, email, phone, message *string, partnerID, referralID *int64) (Lead, error) {
	var l Lead
	err := r.conn().QueryRow(ctx, `
		INSERT INTO crm.lead (code, contact_name, business_name, email, phone, message, status, partner_id, referral_id)
		VALUES ($1, $2, $3, $4, $5, $6, 'new', $7, $8)
		RETURNING `+leadColumns+`
	`, code, contactName, business, email, phone, message, partnerID, referralID).Scan(scanLead(&l)...)
	return l, err
}

func (r *CRMRepository) GetLeadByID(ctx context.Context, id int64) (Lead, error) {
	var l Lead
	err := r.conn().QueryRow(ctx, `
		SELECT `+leadColumns+`
		FROM crm.lead WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(scanLead(&l)...)
	return l, err
}

func (r *CRMRepository) GetLeadByIDForUpdate(ctx context.Context, id int64) (Lead, error) {
	var l Lead
	err := r.conn().QueryRow(ctx, `
		SELECT `+leadColumns+`
		FROM crm.lead WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, id).Scan(scanLead(&l)...)
	return l, err
}

func (r *CRMRepository) ListLeads(ctx context.Context, status string, limit, offset int) ([]Lead, error) {
	query := `SELECT ` + leadColumns + ` FROM crm.lead WHERE deleted_at IS NULL`
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

	var out []Lead
	for rows.Next() {
		var l Lead
		if err := rows.Scan(scanLead(&l)...); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *CRMRepository) CountLeads(ctx context.Context, status string) (int, error) {
	query := `SELECT COUNT(*) FROM crm.lead WHERE deleted_at IS NULL`
	args := []any{}
	if status != "" {
		args = append(args, status)
		query += ` AND status = $1::varchar`
	}
	var n int
	err := r.conn().QueryRow(ctx, query, args...).Scan(&n)
	return n, err
}

func (r *CRMRepository) AssignLead(ctx context.Context, id, assignee int64) (Lead, error) {
	var l Lead
	err := r.conn().QueryRow(ctx, `
		UPDATE crm.lead
		SET assigned_to = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+leadColumns+`
	`, id, assignee).Scan(scanLead(&l)...)
	return l, err
}

func (r *CRMRepository) UpdateLead(ctx context.Context, id int64, status *string, notes *string, buyerID *int64) (Lead, error) {
	setClause := `updated_at = NOW()`
	args := []any{id}
	if status != nil {
		args = append(args, *status)
		setClause += `, status = $` + strconv.Itoa(len(args)) + `::varchar`
	}
	if notes != nil {
		args = append(args, *notes)
		setClause += `, notes = $` + strconv.Itoa(len(args))
	}
	if buyerID != nil {
		args = append(args, *buyerID)
		setClause += `, buyer_id = $` + strconv.Itoa(len(args))
	}
	var l Lead
	err := r.conn().QueryRow(ctx, `
		UPDATE crm.lead SET `+setClause+`
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+leadColumns+`
	`, args...).Scan(scanLead(&l)...)
	return l, err
}
