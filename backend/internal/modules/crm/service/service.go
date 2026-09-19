package service

// M4 CRM slice 1: partners, referral codes, leads with assignment and
// buyer conversion. Attribution is derived (lead counts), never stored.
// Lead machine: new -> assigned -> contacted -> converted | closed, with
// closed reachable as an early exit from new/assigned/contacted.

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/crm/repository"
	"github.com/atlas-platform/backend/internal/modules/crm/schema"
	"github.com/jackc/pgx/v5"
)

var (
	ErrPartnerNotFound = errors.New("partner not found")
	ErrReferralInvalid = errors.New("invalid partner reference")
	ErrLeadNotFound    = errors.New("lead not found")
	ErrLeadState       = errors.New("lead is not actionable in its state")
	ErrInvalidLead     = errors.New("invalid lead request")
	ErrInvalidPartner  = errors.New("invalid partner request")
	ErrInvalidReferral = errors.New("invalid referral request")
	ErrDuplicate       = errors.New("duplicate code or slug")
)

const (
	LeadNew       = "new"
	LeadAssigned  = "assigned"
	LeadContacted = "contacted"
	LeadConverted = "converted"
	LeadClosed    = "closed"
)

var validLeadTransitions = map[string][]string{
	LeadNew:       {LeadAssigned, LeadClosed},
	LeadAssigned:  {LeadContacted, LeadClosed},
	LeadContacted: {LeadConverted, LeadClosed},
	LeadConverted: {},
	LeadClosed:    {},
}

type CRMService struct {
	db   *database.DB
	repo *repository.CRMRepository
}

func NewCRMService(db *database.DB) *CRMService {
	return &CRMService{db: db, repo: repository.NewCRMRepository(db)}
}

func newCode(prefix string) string {
	return fmt.Sprintf("%s%s%s",
		prefix,
		strconv.FormatInt(time.Now().UnixNano(), 36),
		strconv.FormatUint(uint64(rand.Uint32()), 36),
	)
}

var slugSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	s := slugSanitizer.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "partner"
	}
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}

func isDuplicate(err error) bool {
	return err != nil && database.IsUniqueViolation(err)
}

// CreatePartner registers a partner; slug auto-derives from the name with
// a short suffix on conflict.
func (s *CRMService) CreatePartner(ctx context.Context, req schema.CreatePartnerRequest) (*schema.PartnerResponse, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrInvalidPartner
	}
	slug := strings.TrimSpace(req.Slug)
	if slug == "" {
		slug = slugify(req.Name)
	}
	p, err := s.repo.CreatePartner(ctx, newCode("prt_"), req.Name, slug)
	if err != nil {
		if isDuplicate(err) {
			// Slug collision: single deterministic retry with suffix.
			suffixed := fmt.Sprintf("%s-%s", slug, strconv.FormatInt(time.Now().UnixNano(), 36))
			p, err = s.repo.CreatePartner(ctx, newCode("prt_"), req.Name, suffixed)
			if err != nil {
				if isDuplicate(err) {
					return nil, ErrDuplicate
				}
				return nil, fmt.Errorf("create partner: %w", err)
			}
		} else {
			return nil, fmt.Errorf("create partner: %w", err)
		}
	}
	return toPartnerResponse(p), nil
}

func (s *CRMService) ListPartners(ctx context.Context, limit, offset int) ([]schema.PartnerResponse, int, error) {
	partners, err := s.repo.ListPartners(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list partners: %w", err)
	}
	total, err := s.repo.CountPartners(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count partners: %w", err)
	}
	out := make([]schema.PartnerResponse, 0, len(partners))
	for _, p := range partners {
		out = append(out, *toPartnerResponse(p))
	}
	return out, total, nil
}

// CreateReferral mints a referral code for a partner (auto-generated unless
// an explicit code is supplied).
func (s *CRMService) CreateReferral(ctx context.Context, req schema.CreateReferralRequest) (*schema.ReferralResponse, error) {
	if _, err := s.GetPartner(ctx, req.PartnerID); err != nil {
		return nil, err
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		code = "REF-" + strings.ToUpper(strconv.FormatUint(uint64(rand.Uint32()), 36))
	}
	rc, err := s.repo.CreateReferral(ctx, code, req.PartnerID)
	if err != nil {
		if isDuplicate(err) {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("create referral: %w", err)
	}
	return s.toReferralResponse(ctx, rc)
}

func (s *CRMService) GetPartner(ctx context.Context, id int64) (*schema.PartnerResponse, error) {
	p, err := s.repo.GetPartnerByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPartnerNotFound
		}
		return nil, fmt.Errorf("get partner: %w", err)
	}
	return toPartnerResponse(p), nil
}

func (s *CRMService) ListReferrals(ctx context.Context, limit, offset int) ([]schema.ReferralResponse, int, error) {
	codes, err := s.repo.ListReferrals(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list referrals: %w", err)
	}
	total, err := s.repo.CountReferrals(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count referrals: %w", err)
	}
	out := make([]schema.ReferralResponse, 0, len(codes))
	for _, rc := range codes {
		resp, err := s.toReferralResponse(ctx, rc)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *resp)
	}
	return out, total, nil
}

func (s *CRMService) toReferralResponse(ctx context.Context, rc repository.ReferralCode) (*schema.ReferralResponse, error) {
	uses, err := s.repo.ReferralUses(ctx, rc.ID)
	if err != nil {
		return nil, fmt.Errorf("referral uses: %w", err)
	}
	return &schema.ReferralResponse{
		Code: rc.Code, PartnerID: rc.PartnerID, IsActive: rc.IsActive,
		Uses: uses, CreatedAt: rc.CreatedAt,
	}, nil
}

// TrackPartner resolves a campaign reference (referral code or partner
// slug) to its partner. Unknown references are 404, never guessed.
func (s *CRMService) TrackPartner(ctx context.Context, ref string) (*schema.TrackResponse, error) {
	if strings.TrimSpace(ref) == "" {
		return nil, ErrReferralInvalid
	}
	p, _, err := s.repo.ResolvePartnerRef(ctx, ref)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReferralInvalid
		}
		return nil, fmt.Errorf("resolve partner: %w", err)
	}
	return &schema.TrackResponse{Partner: *toPartnerResponse(p)}, nil
}

// SubmitLead records a public enquiry, attributing an optional partner
// reference. Unknown references are rejected with a reason (never silently
// dropped, never guessed).
func (s *CRMService) SubmitLead(ctx context.Context, req schema.SubmitLeadRequest) (*schema.LeadResponse, error) {
	if strings.TrimSpace(req.ContactName) == "" || strings.TrimSpace(req.BusinessName) == "" {
		return nil, ErrInvalidLead
	}
	var partnerID, referralID *int64
	if strings.TrimSpace(req.PartnerRef) != "" {
		p, rc, err := s.repo.ResolvePartnerRef(ctx, req.PartnerRef)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrReferralInvalid
			}
			return nil, fmt.Errorf("resolve partner: %w", err)
		}
		partnerID = &p.ID
		if rc != nil {
			referralID = &rc.ID
		}
	}
	l, err := s.repo.CreateLead(ctx, newCode("lead_"), req.ContactName, req.BusinessName,
		emptyToNil(req.Email), emptyToNil(req.Phone), emptyToNil(req.Message), partnerID, referralID)
	if err != nil {
		return nil, fmt.Errorf("create lead: %w", err)
	}
	return toLeadResponse(l), nil
}

func emptyToNil(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

// AssignLead moves new → assigned and records the staffer, atomically:
// assignment and status move commit together or not at all.
func (s *CRMService) AssignLead(ctx context.Context, id, assignee int64) (*schema.LeadResponse, error) {
	if assignee <= 0 {
		return nil, ErrInvalidLead
	}
	var updated repository.Lead
	err := s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCRMRepositoryWithTx(tx)
		locked, err := txRepo.GetLeadByIDForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrLeadNotFound
			}
			return fmt.Errorf("lock lead: %w", err)
		}
		if locked.Status != LeadNew {
			return ErrLeadState
		}
		if _, err := txRepo.AssignLead(ctx, locked.ID, assignee); err != nil {
			return fmt.Errorf("assign lead: %w", err)
		}
		to := LeadAssigned
		u, err := txRepo.UpdateLead(ctx, locked.ID, &to, nil, nil)
		if err != nil {
			return fmt.Errorf("move lead to assigned: %w", err)
		}
		updated = u
		return nil
	})
	if err != nil {
		return nil, err
	}
	return toLeadResponse(updated), nil
}

// UpdateLead applies notes and/or a status move with transition rules. A
// move to converted requires buyer_id (FK-guarded: unknown buyers fail).
func (s *CRMService) UpdateLead(ctx context.Context, id int64, req schema.UpdateLeadRequest) (*schema.LeadResponse, error) {
	var updated repository.Lead
	err := s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCRMRepositoryWithTx(tx)
		locked, err := txRepo.GetLeadByIDForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrLeadNotFound
			}
			return fmt.Errorf("lock lead: %w", err)
		}
		if req.Status != nil && *req.Status != locked.Status {
			if !validLeadTransition(locked.Status, *req.Status) {
				return ErrLeadState
			}
			if *req.Status == LeadConverted && req.BuyerID == nil {
				return ErrInvalidLead
			}
		}
		u, err := txRepo.UpdateLead(ctx, locked.ID, req.Status, req.Notes, req.BuyerID)
		if err != nil {
			if database.IsForeignKeyViolation(err) {
				return ErrInvalidLead
			}
			return fmt.Errorf("update lead: %w", err)
		}
		updated = u
		return nil
	})
	if err != nil {
		return nil, err
	}
	return toLeadResponse(updated), nil
}

func validLeadTransition(from, to string) bool {
	for _, next := range validLeadTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

func (s *CRMService) GetLead(ctx context.Context, id int64) (*schema.LeadResponse, error) {
	l, err := s.repo.GetLeadByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLeadNotFound
		}
		return nil, fmt.Errorf("get lead: %w", err)
	}
	return toLeadResponse(l), nil
}

func (s *CRMService) ListLeads(ctx context.Context, status string, limit, offset int) ([]schema.LeadResponse, int, error) {
	leads, err := s.repo.ListLeads(ctx, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list leads: %w", err)
	}
	total, err := s.repo.CountLeads(ctx, status)
	if err != nil {
		return nil, 0, fmt.Errorf("count leads: %w", err)
	}
	out := make([]schema.LeadResponse, 0, len(leads))
	for _, l := range leads {
		out = append(out, *toLeadResponse(l))
	}
	return out, total, nil
}

func toPartnerResponse(p repository.Partner) *schema.PartnerResponse {
	return &schema.PartnerResponse{
		Code: p.Code, Name: p.Name, Slug: p.Slug, IsActive: p.IsActive,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toLeadResponse(l repository.Lead) *schema.LeadResponse {
	out := &schema.LeadResponse{
		Code: l.Code, ContactName: l.ContactName, BusinessName: l.Business,
		Status: l.Status, PartnerID: l.PartnerID, ReferralID: l.ReferralID,
		AssignedTo: l.AssignedTo, BuyerID: l.BuyerID,
		CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt,
	}
	if l.Email != nil {
		out.Email = *l.Email
	}
	if l.Phone != nil {
		out.Phone = *l.Phone
	}
	if l.Message != nil {
		out.Message = *l.Message
	}
	out.Notes = l.Notes
	return out
}
