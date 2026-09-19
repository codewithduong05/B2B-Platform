package service

// M4 suppliers slice 1: operational profiles, application, approval,
// versioned contracts. Supplier fulfilment (order ack) and stock push are
// deferred: they need order-status and inventory-write semantics decided
// first and are explicitly out of scope.

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	catalog_repo "github.com/atlas-platform/backend/internal/modules/catalog/repository"
	commerce_repo "github.com/atlas-platform/backend/internal/modules/commerce/repository"
	commerce_service "github.com/atlas-platform/backend/internal/modules/commerce/service"
	inventory_service "github.com/atlas-platform/backend/internal/modules/inventory/service"
	"github.com/atlas-platform/backend/internal/modules/suppliers/repository"
	"github.com/atlas-platform/backend/internal/modules/suppliers/schema"
	"github.com/jackc/pgx/v5"
)

var (
	ErrSupplierNotFound = errors.New("supplier not found")
	ErrSupplierState    = errors.New("supplier is not actionable in its state")
	ErrInvalidSupplier  = errors.New("invalid supplier request")
	ErrInvalidContract  = errors.New("invalid contract request")
	ErrDuplicate        = errors.New("duplicate supplier code")
)

const (
	SupplierApplied  = "applied"
	SupplierApproved = "approved"
)

type SupplierService struct {
	db                  *database.DB
	repo                *repository.SupplierRepository
	catalogSupplierRepo *catalog_repo.SupplierRepository
	catalogProductRepo  *catalog_repo.ProductRepository
	commerceRepo        *commerce_repo.CommerceRepository
	commerceSvc         *commerce_service.CommerceService
	inventorySvc        *inventory_service.InventoryService
}

func NewSupplierService(db *database.DB) *SupplierService {
	return &SupplierService{db: db, repo: repository.NewSupplierRepository(db)}
}

func newCode(prefix string) string {
	return fmt.Sprintf("%s%s%s",
		prefix,
		strconv.FormatInt(time.Now().UnixNano(), 36),
		strconv.FormatUint(uint64(rand.Uint32()), 36),
	)
}

func parseTimePtr(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func strPtr(s *string) *string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	v := strings.TrimSpace(*s)
	return &v
}

func optStr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	v := strings.TrimSpace(s)
	return &v
}

// trimmedOrNil maps update semantics: nil means untouched; otherwise the
// trimmed value, or NULL when allowEmpty clears the field.
func trimmedOrNil(s *string, allowEmpty bool) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" && !allowEmpty {
		return &v
	}
	if v == "" {
		return nil
	}
	return &v
}

// Apply records a supplier application. Authenticated applicants are linked
// automatically (portal login); anonymous applications stay unlinked until
// staff attaches a user at approval.
func (s *SupplierService) Apply(ctx context.Context, userID int64, req schema.ApplySupplierRequest) (*schema.SupplierResponse, error) {
	if strings.TrimSpace(req.CompanyName) == "" {
		return nil, ErrInvalidSupplier
	}
	var linked *int64
	if userID > 0 {
		linked = &userID
	}
	p, err := s.repo.CreateProfile(ctx, newCode("sup_"), req.CompanyName,
		optStr(req.ContactName), optStr(req.ContactEmail), optStr(req.ContactPhone), linked)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("create supplier profile: %w", err)
	}
	return toSupplierResponse(p), nil
}

// Approve moves applied → approved, optionally linking a user. The
// conditional update makes concurrent approves safe: exactly one wins,
// losers see no row and get a state error.
func (s *SupplierService) Approve(ctx context.Context, actor, id int64, userID *int64) (*schema.SupplierResponse, error) {
	if userID != nil && *userID <= 0 {
		return nil, ErrInvalidSupplier
	}
	updated, err := s.repo.ApproveProfile(ctx, id, actor, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			existing, gerr := s.repo.GetProfileByID(ctx, id)
			if gerr != nil {
				return nil, ErrSupplierNotFound
			}
			if existing.Status != SupplierApplied {
				return nil, ErrSupplierState
			}
			return nil, ErrSupplierState
		}
		if database.IsForeignKeyViolation(err) {
			return nil, ErrInvalidSupplier
		}
		return nil, fmt.Errorf("approve supplier: %w", err)
	}
	return toSupplierResponse(updated), nil
}

func (s *SupplierService) GetSupplier(ctx context.Context, id int64) (*schema.SupplierResponse, error) {
	p, err := s.repo.GetProfileByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSupplierNotFound
		}
		return nil, fmt.Errorf("get supplier: %w", err)
	}
	return toSupplierResponse(p), nil
}

// MyProfile resolves the caller's operational profile. Unlinked principals
// get 404 (never 403, per the information-leak rule).
func (s *SupplierService) MyProfile(ctx context.Context, userID int64) (*schema.SupplierResponse, error) {
	p, err := s.repo.GetProfileByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSupplierNotFound
		}
		return nil, fmt.Errorf("get supplier profile: %w", err)
	}
	return toSupplierResponse(p), nil
}

// UpdateMyProfile edits the caller's own contact fields.
func (s *SupplierService) UpdateMyProfile(ctx context.Context, userID int64, req schema.UpdateSupplierRequest) (*schema.SupplierResponse, error) {
	p, err := s.repo.GetProfileByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSupplierNotFound
		}
		return nil, fmt.Errorf("get supplier profile: %w", err)
	}
	if req.CompanyName != nil && strings.TrimSpace(*req.CompanyName) == "" {
		return nil, ErrInvalidSupplier
	}
	updated, err := s.repo.UpdateProfile(ctx, p.ID,
		trimmedOrNil(req.CompanyName, false),
		trimmedOrNil(req.ContactName, true),
		trimmedOrNil(req.ContactEmail, true),
		trimmedOrNil(req.ContactPhone, true))
	if err != nil {
		return nil, fmt.Errorf("update supplier profile: %w", err)
	}
	return toSupplierResponse(updated), nil
}

func (s *SupplierService) ListSuppliers(ctx context.Context, status string, limit, offset int) ([]schema.SupplierResponse, int, error) {
	profiles, err := s.repo.ListProfiles(ctx, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list suppliers: %w", err)
	}
	total, err := s.repo.CountProfiles(ctx, status)
	if err != nil {
		return nil, 0, fmt.Errorf("count suppliers: %w", err)
	}
	out := make([]schema.SupplierResponse, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, *toSupplierResponse(p))
	}
	return out, total, nil
}

// SetContract mints a new immutable contract version and deactivates the
// previous current one, atomically.
func (s *SupplierService) SetContract(ctx context.Context, actor, supplierID int64, req schema.SetContractRequest) (*schema.ContractResponse, error) {
	if strings.TrimSpace(req.Terms) == "" {
		return nil, ErrInvalidContract
	}
	validFrom, err := parseTimePtr(req.ValidFrom)
	if err != nil {
		return nil, ErrInvalidContract
	}
	validTo, err := parseTimePtr(req.ValidTo)
	if err != nil {
		return nil, ErrInvalidContract
	}
	if validFrom != nil && validTo != nil && validTo.Before(*validFrom) {
		return nil, ErrInvalidContract
	}
	if _, err := s.GetSupplier(ctx, supplierID); err != nil {
		return nil, err
	}
	var created repository.SupplierContract
	err = s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewSupplierRepositoryWithTx(tx)
		version, err := txRepo.CurrentContractVersion(ctx, supplierID)
		if err != nil {
			return fmt.Errorf("contract version: %w", err)
		}
		if err := txRepo.DeactivateCurrentContracts(ctx, supplierID); err != nil {
			return fmt.Errorf("deactivate contracts: %w", err)
		}
		c, err := txRepo.CreateContractVersion(ctx, supplierID, int64(version+1), req.Terms, validFrom, validTo, &actor)
		if err != nil {
			return fmt.Errorf("create contract: %w", err)
		}
		created = c
		return nil
	})
	if err != nil {
		return nil, err
	}
	return toContractResponse(created), nil
}

func (s *SupplierService) GetContracts(ctx context.Context, supplierID int64) ([]schema.ContractResponse, error) {
	if _, err := s.GetSupplier(ctx, supplierID); err != nil {
		return nil, err
	}
	contracts, err := s.repo.GetContracts(ctx, supplierID)
	if err != nil {
		return nil, fmt.Errorf("get contracts: %w", err)
	}
	out := make([]schema.ContractResponse, 0, len(contracts))
	for _, c := range contracts {
		out = append(out, *toContractResponse(c))
	}
	return out, nil
}

func toSupplierResponse(p repository.SupplierProfile) *schema.SupplierResponse {
	out := &schema.SupplierResponse{
		Code: p.Code, CompanyName: p.CompanyName, Status: p.Status,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
	if p.ContactName != nil {
		out.ContactName = *p.ContactName
	}
	return out
}

func toContractResponse(c repository.SupplierContract) *schema.ContractResponse {
	return &schema.ContractResponse{
		Version: c.Version, Terms: c.Terms, ValidFrom: c.ValidFrom,
		ValidTo: c.ValidTo, IsCurrent: c.IsCurrent, CreatedAt: c.CreatedAt,
	}
}
