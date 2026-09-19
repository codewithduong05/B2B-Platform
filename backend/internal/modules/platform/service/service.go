package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/platform/repository"
)

var (
	ErrNotFound          = errors.New("platform: not found")
	ErrInvalidInput      = errors.New("platform: invalid input")
	ErrTemplateInactive  = errors.New("platform: template inactive")
	ErrRecipientNotFound = errors.New("platform: recipient not found")
)

type PlatformService struct {
	repo *repository.PlatformRepository
}

func NewPlatformService(db *database.DB) *PlatformService {
	return &PlatformService{
		repo: repository.NewPlatformRepository(db),
	}
}

func (s *PlatformService) CreateTemplate(ctx context.Context, code, name, channel string, subject *string, body string, variables []string, createdBy *int64) (*repository.NotificationTemplate, error) {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	channel = strings.TrimSpace(channel)

	if code == "" || name == "" || channel == "" || body == "" {
		return nil, ErrInvalidInput
	}
	if channel != "email" && channel != "sms" {
		return nil, ErrInvalidInput
	}
	if channel == "email" && (subject == nil || *subject == "") {
		return nil, ErrInvalidInput
	}

	return s.repo.CreateTemplate(ctx, code, name, channel, subject, body, variables, createdBy)
}

func (s *PlatformService) GetTemplateByCode(ctx context.Context, code string) (*repository.NotificationTemplate, error) {
	t, err := s.repo.GetTemplateByCode(ctx, code)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return t, nil
}

func (s *PlatformService) ListTemplates(ctx context.Context, channel, status string, limit, offset int) ([]repository.NotificationTemplate, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListTemplates(ctx, channel, status, limit, offset)
}

func (s *PlatformService) UpdateTemplate(ctx context.Context, id int64, name *string, subject *string, body *string, variables []string, status *string) (*repository.NotificationTemplate, error) {
	if status != nil && *status != "active" && *status != "inactive" {
		return nil, ErrInvalidInput
	}
	t, err := s.repo.UpdateTemplate(ctx, id, name, subject, body, variables, status)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return t, nil
}

func (s *PlatformService) SendNotification(ctx context.Context, templateCode, recipientType string, recipientID int64, variables map[string]interface{}) (*repository.Notification, error) {
	templateCode = strings.TrimSpace(templateCode)
	recipientType = strings.TrimSpace(recipientType)

	if templateCode == "" || recipientType == "" || recipientID <= 0 {
		return nil, ErrInvalidInput
	}
	if recipientType != "buyer" && recipientType != "user" {
		return nil, ErrInvalidInput
	}

	template, err := s.repo.GetTemplateByCode(ctx, templateCode)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if template.Status != "active" {
		return nil, ErrTemplateInactive
	}

	var address string
	switch recipientType {
	case "buyer":
		address, err = s.repo.GetBuyerEmail(ctx, recipientID)
	case "user":
		address, err = s.repo.GetUserEmail(ctx, recipientID)
	}
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrRecipientNotFound
		}
		return nil, err
	}

	subject := renderTemplate(template.Subject, variables)
	body := renderTemplate(&template.Body, variables)

	code := generateNotificationCode()
	notification, err := s.repo.CreateNotification(ctx, code, &template.ID, recipientType, recipientID, template.Channel, address, subject, *body)
	if err != nil {
		return nil, err
	}

	_ = s.repo.CreateNotificationEvent(ctx, notification.ID, "created", nil, nil)

	if err := s.deliverNotification(ctx, notification, template.Channel); err != nil {
		_ = s.repo.UpdateNotificationStatus(ctx, notification.ID, "failed", nil, strPtr(err.Error()))
		_ = s.repo.CreateNotificationEvent(ctx, notification.ID, "failed", nil, strPtr(err.Error()))
		return notification, nil
	}

	providerMsgID := fmt.Sprintf("sim_%d_%d", notification.ID, time.Now().UnixNano())
	_ = s.repo.UpdateNotificationStatus(ctx, notification.ID, "sent", &providerMsgID, nil)
	_ = s.repo.CreateNotificationEvent(ctx, notification.ID, "sent", &providerMsgID, nil)

	notification.Status = "sent"
	notification.ProviderMessageID = &providerMsgID
	notification.SentAt = timePtr(time.Now())
	return notification, nil
}

func (s *PlatformService) deliverNotification(ctx context.Context, notification *repository.Notification, channel string) error {
	return nil
}

func (s *PlatformService) ListBuyerNotifications(ctx context.Context, buyerID int64, status string, limit, offset int) ([]repository.Notification, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListNotifications(ctx, "buyer", buyerID, status, limit, offset)
}

func (s *PlatformService) ListAdminNotifications(ctx context.Context, channel, status string, limit, offset int) ([]repository.Notification, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListAdminNotifications(ctx, channel, status, limit, offset)
}

func (s *PlatformService) GetNotificationByCode(ctx context.Context, code string) (*repository.Notification, error) {
	n, err := s.repo.GetNotificationByCode(ctx, code)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return n, nil
}

func (s *PlatformService) GetNotificationEvents(ctx context.Context, notificationID int64) ([]repository.NotificationEvent, error) {
	return s.repo.ListNotificationEvents(ctx, notificationID)
}

func (s *PlatformService) CreateMediaUpload(ctx context.Context, filename, contentType string, createdBy *int64) (*repository.MediaUpload, error) {
	filename = strings.TrimSpace(filename)
	contentType = strings.TrimSpace(contentType)

	if filename == "" || contentType == "" {
		return nil, ErrInvalidInput
	}

	code := fmt.Sprintf("media_%d", time.Now().UnixNano())
	uploadURL := fmt.Sprintf("https://storage.example.com/upload/%s", code)
	expiresAt := time.Now().Add(1 * time.Hour)

	return s.repo.CreateMediaUpload(ctx, code, filename, contentType, uploadURL, expiresAt, createdBy)
}

func (s *PlatformService) GetMediaUploadByCode(ctx context.Context, code string) (*repository.MediaUpload, error) {
	m, err := s.repo.GetMediaUploadByCode(ctx, code)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return m, nil
}

func (s *PlatformService) ListActiveRegions(ctx context.Context) ([]repository.ReferenceRegion, error) {
	return s.repo.ListActiveRegions(ctx)
}

func (s *PlatformService) ListFeatureFlags(ctx context.Context, enabled *bool, limit, offset int) ([]repository.FeatureFlag, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListFeatureFlags(ctx, enabled, limit, offset)
}

func (s *PlatformService) GetFeatureFlagByKey(ctx context.Context, key string) (*repository.FeatureFlag, error) {
	f, err := s.repo.GetFeatureFlagByKey(ctx, key)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

func (s *PlatformService) UpsertFeatureFlag(ctx context.Context, key, name string, description *string, enabled bool, updatedBy *int64) (*repository.FeatureFlag, error) {
	key = strings.TrimSpace(key)
	name = strings.TrimSpace(name)

	if key == "" || name == "" {
		return nil, ErrInvalidInput
	}

	return s.repo.UpsertFeatureFlag(ctx, key, name, description, enabled, updatedBy)
}

func (s *PlatformService) ListAuditLog(ctx context.Context, actorType, action, resourceType string, actorID, resourceID *int64, limit, offset int) ([]repository.AuditLog, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListAuditLog(ctx, actorType, action, resourceType, actorID, resourceID, limit, offset)
}

func (s *PlatformService) ListIntegrationTraffic(ctx context.Context, direction, provider string, limit, offset int) ([]repository.IntegrationTraffic, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListIntegrationTraffic(ctx, direction, provider, limit, offset)
}

var templateVarRegex = regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)

func renderTemplate(template *string, variables map[string]interface{}) *string {
	if template == nil {
		return nil
	}
	result := templateVarRegex.ReplaceAllStringFunc(*template, func(match string) string {
		submatches := templateVarRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		key := submatches[1]
		if val, ok := variables[key]; ok {
			return fmt.Sprintf("%v", val)
		}
		return match
	})
	return &result
}

func generateNotificationCode() string {
	return fmt.Sprintf("notif_%d", time.Now().UnixNano())
}

func strPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}
