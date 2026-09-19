package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/ai/repository"
)

var (
	ErrNotFound        = errors.New("ai: not found")
	ErrInvalidInput    = errors.New("ai: invalid input")
	ErrProposalExpired = errors.New("ai: proposal expired")
	ErrBudgetExceeded  = errors.New("ai: budget exceeded")
)

type AIService struct {
	repo *repository.AIRepository
}

func NewAIService(db *database.DB) *AIService {
	return &AIService{
		repo: repository.NewAIRepository(db),
	}
}

func (s *AIService) CreatePrompt(ctx context.Context, key, name string, description, owner *string) (*repository.Prompt, error) {
	key = strings.TrimSpace(key)
	name = strings.TrimSpace(name)

	if key == "" || name == "" {
		return nil, ErrInvalidInput
	}

	return s.repo.CreatePrompt(ctx, key, name, description, owner)
}

func (s *AIService) GetPromptByKey(ctx context.Context, key string) (*repository.Prompt, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, ErrInvalidInput
	}

	p, err := s.repo.GetPromptByKey(ctx, key)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (s *AIService) ListPrompts(ctx context.Context, limit, offset int) ([]repository.Prompt, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListPrompts(ctx, limit, offset)
}

func (s *AIService) CreatePromptVersion(ctx context.Context, promptKey, content string) (*repository.PromptVersion, error) {
	promptKey = strings.TrimSpace(promptKey)
	content = strings.TrimSpace(content)

	if promptKey == "" || content == "" {
		return nil, ErrInvalidInput
	}

	prompt, err := s.repo.GetPromptByKey(ctx, promptKey)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	latest, err := s.repo.GetLatestPromptVersion(ctx, prompt.ID)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return nil, err
	}

	nextVersion := 1
	if latest != nil {
		nextVersion = latest.Version + 1
	}

	return s.repo.CreatePromptVersion(ctx, prompt.ID, nextVersion, content, "draft")
}

func (s *AIService) CreateProposal(ctx context.Context, feature string, confidence float64, items, unresolved interface{}, promptKey string, promptVersion int, model string, inputHash string, expiresAt time.Time, tenantID *int64) (*repository.Proposal, error) {
	feature = strings.TrimSpace(feature)
	promptKey = strings.TrimSpace(promptKey)
	model = strings.TrimSpace(model)

	if feature == "" || promptKey == "" || model == "" {
		return nil, ErrInvalidInput
	}
	if confidence < 0 || confidence > 1 {
		return nil, ErrInvalidInput
	}
	if expiresAt.Before(time.Now()) {
		return nil, ErrInvalidInput
	}

	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return nil, ErrInvalidInput
	}
	unresolvedJSON, err := json.Marshal(unresolved)
	if err != nil {
		return nil, ErrInvalidInput
	}

	provenanceJSON, err := repository.MarshalProvenance(promptKey, promptVersion, model, inputHash)
	if err != nil {
		return nil, ErrInvalidInput
	}

	code := generateCode()

	return s.repo.CreateProposal(ctx, code, feature, confidence, itemsJSON, unresolvedJSON, provenanceJSON, expiresAt, tenantID)
}

func (s *AIService) GetProposalByCode(ctx context.Context, code string) (*repository.Proposal, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, ErrInvalidInput
	}

	p, err := s.repo.GetProposalByCode(ctx, code)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (s *AIService) ConfirmProposal(ctx context.Context, code string) (*repository.Proposal, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, ErrInvalidInput
	}

	proposal, err := s.repo.GetProposalByCode(ctx, code)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if proposal.Status != "proposed" {
		return nil, ErrInvalidInput
	}

	if proposal.ExpiresAt.Before(time.Now()) {
		return nil, ErrProposalExpired
	}

	return s.repo.ConfirmProposal(ctx, code)
}

func (s *AIService) RejectProposal(ctx context.Context, code string) (*repository.Proposal, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, ErrInvalidInput
	}

	proposal, err := s.repo.GetProposalByCode(ctx, code)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if proposal.Status != "proposed" {
		return nil, ErrInvalidInput
	}

	return s.repo.RejectProposal(ctx, code)
}

func (s *AIService) RecordUsage(ctx context.Context, proposalID *int64, promptKey string, promptVersion int, model string, inputTokens, outputTokens, latencyMS int, costMicros int64, tenantID *int64, feature string) (*repository.UsageRecord, error) {
	promptKey = strings.TrimSpace(promptKey)
	model = strings.TrimSpace(model)
	feature = strings.TrimSpace(feature)

	if promptKey == "" || model == "" || feature == "" {
		return nil, ErrInvalidInput
	}
	if inputTokens < 0 || outputTokens < 0 || latencyMS < 0 || costMicros < 0 {
		return nil, ErrInvalidInput
	}

	return s.repo.RecordUsage(ctx, proposalID, promptKey, promptVersion, model, inputTokens, outputTokens, latencyMS, costMicros, tenantID, feature)
}

func (s *AIService) ListUsage(ctx context.Context, feature string, tenantID *int64, limit, offset int) ([]repository.UsageRecord, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListUsage(ctx, feature, tenantID, limit, offset)
}

func (s *AIService) CreateModelRouting(ctx context.Context, featureKey, provider, modelName string, config json.RawMessage) (*repository.ModelRouting, error) {
	featureKey = strings.TrimSpace(featureKey)
	provider = strings.TrimSpace(provider)
	modelName = strings.TrimSpace(modelName)

	if featureKey == "" || provider == "" || modelName == "" {
		return nil, ErrInvalidInput
	}

	var configBytes []byte
	if config != nil {
		configBytes = []byte(config)
	}

	return s.repo.CreateModelRouting(ctx, featureKey, provider, modelName, configBytes)
}

func (s *AIService) GetModelRoutingByFeature(ctx context.Context, featureKey string) (*repository.ModelRouting, error) {
	featureKey = strings.TrimSpace(featureKey)
	if featureKey == "" {
		return nil, ErrInvalidInput
	}

	m, err := s.repo.GetModelRoutingByFeature(ctx, featureKey)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return m, nil
}

func (s *AIService) ListModelRouting(ctx context.Context, limit, offset int) ([]repository.ModelRouting, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListModelRouting(ctx, limit, offset)
}

func (s *AIService) UpdateModelRouting(ctx context.Context, featureKey string, provider, modelName *string, config json.RawMessage) (*repository.ModelRouting, error) {
	featureKey = strings.TrimSpace(featureKey)
	if featureKey == "" {
		return nil, ErrInvalidInput
	}

	var configBytes []byte
	if config != nil {
		configBytes = []byte(config)
	}

	return s.repo.UpdateModelRouting(ctx, featureKey, provider, modelName, configBytes)
}

func (s *AIService) CreateBudget(ctx context.Context, scopeKey string, periodStart, periodEnd time.Time, ceilingMicros int64) (*repository.Budget, error) {
	scopeKey = strings.TrimSpace(scopeKey)

	if scopeKey == "" {
		return nil, ErrInvalidInput
	}
	if ceilingMicros <= 0 {
		return nil, ErrInvalidInput
	}
	if !periodEnd.After(periodStart) {
		return nil, ErrInvalidInput
	}

	return s.repo.CreateBudget(ctx, scopeKey, periodStart, periodEnd, ceilingMicros)
}

func (s *AIService) GetBudgetByScope(ctx context.Context, scopeKey string) (*repository.Budget, error) {
	scopeKey = strings.TrimSpace(scopeKey)
	if scopeKey == "" {
		return nil, ErrInvalidInput
	}

	b, err := s.repo.GetBudgetByScope(ctx, scopeKey)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return b, nil
}

func (s *AIService) ListBudgets(ctx context.Context, limit, offset int) ([]repository.Budget, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListBudgets(ctx, limit, offset)
}

func (s *AIService) CheckBudget(ctx context.Context, scopeKey string, estimatedCostMicros int64) (bool, error) {
	budget, err := s.repo.GetBudgetByScope(ctx, scopeKey)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return true, nil
		}
		return false, err
	}

	return budget.UsedMicros+estimatedCostMicros <= budget.CeilingMicros, nil
}

func generateCode() string {
	hash := sha256.New()
	hash.Write([]byte(time.Now().String()))
	return "PROP-" + hex.EncodeToString(hash.Sum(nil))[:20]
}
