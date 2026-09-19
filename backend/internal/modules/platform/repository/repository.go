package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/atlas-platform/backend/internal/database"
)

type PlatformRepository struct {
	db *database.DB
}

func NewPlatformRepository(db *database.DB) *PlatformRepository {
	return &PlatformRepository{db: db}
}

type NotificationTemplate struct {
	ID        int64
	Code      string
	Name      string
	Channel   string
	Subject   *string
	Body      string
	Variables []string
	Status    string
	CreatedBy *int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (r *PlatformRepository) CreateTemplate(ctx context.Context, code, name, channel string, subject *string, body string, variables []string, createdBy *int64) (*NotificationTemplate, error) {
	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return nil, fmt.Errorf("marshal variables: %w", err)
	}

	var t NotificationTemplate
	err = r.db.Pool.QueryRow(ctx, `
		INSERT INTO platform.notification_template (code, name, channel, subject, body_template, variables, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, code, name, channel, subject, body_template, variables, status, created_at, updated_at
	`, code, name, channel, subject, body, variablesJSON, createdBy).Scan(
		&t.ID, &t.Code, &t.Name, &t.Channel, &t.Subject, &t.Body, &variablesJSON, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(variablesJSON, &t.Variables)
	return &t, nil
}

func (r *PlatformRepository) GetTemplateByCode(ctx context.Context, code string) (*NotificationTemplate, error) {
	var t NotificationTemplate
	var variablesJSON []byte
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, code, name, channel, subject, body_template, variables, status, created_at, updated_at
		FROM platform.notification_template
		WHERE code = $1 AND deleted_at IS NULL
	`, code).Scan(&t.ID, &t.Code, &t.Name, &t.Channel, &t.Subject, &t.Body, &variablesJSON, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(variablesJSON, &t.Variables)
	return &t, nil
}

func (r *PlatformRepository) ListTemplates(ctx context.Context, channel, status string, limit, offset int) ([]NotificationTemplate, int, error) {
	var total int
	err := r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM platform.notification_template
		WHERE deleted_at IS NULL AND ($1 = '' OR channel = $1) AND ($2 = '' OR status = $2)
	`, channel, status).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, code, name, channel, subject, body_template, variables, status, created_at, updated_at
		FROM platform.notification_template
		WHERE deleted_at IS NULL AND ($1 = '' OR channel = $1) AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC, id DESC
		LIMIT $3 OFFSET $4
	`, channel, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []NotificationTemplate
	for rows.Next() {
		var t NotificationTemplate
		var variablesJSON []byte
		if err := rows.Scan(&t.ID, &t.Code, &t.Name, &t.Channel, &t.Subject, &t.Body, &variablesJSON, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		_ = json.Unmarshal(variablesJSON, &t.Variables)
		out = append(out, t)
	}
	return out, total, rows.Err()
}

func (r *PlatformRepository) UpdateTemplate(ctx context.Context, id int64, name *string, subject *string, body *string, variables []string, status *string) (*NotificationTemplate, error) {
	var variablesJSON []byte
	if variables != nil {
		var err error
		variablesJSON, err = json.Marshal(variables)
		if err != nil {
			return nil, fmt.Errorf("marshal variables: %w", err)
		}
	}

	var t NotificationTemplate
	var vj []byte
	err := r.db.Pool.QueryRow(ctx, `
		UPDATE platform.notification_template
		SET name = COALESCE($2, name),
		    subject = COALESCE($3, subject),
		    body_template = COALESCE($4, body_template),
		    variables = COALESCE($5, variables),
		    status = COALESCE($6, status),
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, code, name, channel, subject, body_template, variables, status, created_at, updated_at
	`, id, name, subject, body, variablesJSON, status).Scan(
		&t.ID, &t.Code, &t.Name, &t.Channel, &t.Subject, &t.Body, &vj, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(vj, &t.Variables)
	return &t, nil
}

type Notification struct {
	ID                int64
	Code              string
	TemplateID        *int64
	TemplateCode      *string
	RecipientType     string
	RecipientID       int64
	Channel           string
	RecipientAddress  string
	Subject           *string
	Body              string
	Status            string
	ProviderMessageID *string
	SentAt            *time.Time
	DeliveredAt       *time.Time
	FailedAt          *time.Time
	ErrorMessage      *string
	RetryCount        int
	CreatedAt         time.Time
}

func (r *PlatformRepository) CreateNotification(ctx context.Context, code string, templateID *int64, recipientType string, recipientID int64, channel, address string, subject *string, body string) (*Notification, error) {
	var n Notification
	var templateCode pgtype.Text
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO platform.notification (code, template_id, recipient_type, recipient_id, channel, recipient_address, subject, body)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, code, template_id, 
		          (SELECT code FROM platform.notification_template WHERE id = $2),
		          recipient_type, recipient_id, channel, recipient_address, subject, body,
		          status, provider_message_id, sent_at, delivered_at, failed_at, error_message, retry_count, created_at
	`, code, templateID, recipientType, recipientID, channel, address, subject, body).Scan(
		&n.ID, &n.Code, &n.TemplateID, &templateCode, &n.RecipientType, &n.RecipientID,
		&n.Channel, &n.RecipientAddress, &n.Subject, &n.Body, &n.Status, &n.ProviderMessageID,
		&n.SentAt, &n.DeliveredAt, &n.FailedAt, &n.ErrorMessage, &n.RetryCount, &n.CreatedAt)
	if err != nil {
		return nil, err
	}
	if templateCode.Valid {
		n.TemplateCode = &templateCode.String
	}
	return &n, nil
}

func (r *PlatformRepository) UpdateNotificationStatus(ctx context.Context, id int64, status string, providerMessageID *string, errorMessage *string) error {
	var query string
	switch status {
	case "sent":
		query = `UPDATE platform.notification SET status = $2, provider_message_id = $3, sent_at = NOW(), updated_at = NOW() WHERE id = $1`
	case "delivered":
		query = `UPDATE platform.notification SET status = $2, delivered_at = NOW(), updated_at = NOW() WHERE id = $1`
	case "failed":
		query = `UPDATE platform.notification SET status = $2, error_message = $3, failed_at = NOW(), retry_count = retry_count + 1, updated_at = NOW() WHERE id = $1`
	default:
		query = `UPDATE platform.notification SET status = $2, updated_at = NOW() WHERE id = $1`
	}
	_, err := r.db.Pool.Exec(ctx, query, id, status, providerMessageID, errorMessage)
	return err
}

func (r *PlatformRepository) ListNotifications(ctx context.Context, recipientType string, recipientID int64, status string, limit, offset int) ([]Notification, int, error) {
	var total int
	err := r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM platform.notification
		WHERE recipient_type = $1 AND recipient_id = $2 AND ($3 = '' OR status = $3)
	`, recipientType, recipientID, status).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, code, template_id,
		       (SELECT code FROM platform.notification_template WHERE id = template_id),
		       recipient_type, recipient_id, channel, recipient_address, subject, body,
		       status, provider_message_id, sent_at, delivered_at, failed_at, error_message, retry_count, created_at
		FROM platform.notification
		WHERE recipient_type = $1 AND recipient_id = $2 AND ($3 = '' OR status = $3)
		ORDER BY created_at DESC, id DESC
		LIMIT $4 OFFSET $5
	`, recipientType, recipientID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Notification
	for rows.Next() {
		var n Notification
		var templateCode pgtype.Text
		if err := rows.Scan(&n.ID, &n.Code, &n.TemplateID, &templateCode, &n.RecipientType, &n.RecipientID,
			&n.Channel, &n.RecipientAddress, &n.Subject, &n.Body, &n.Status, &n.ProviderMessageID,
			&n.SentAt, &n.DeliveredAt, &n.FailedAt, &n.ErrorMessage, &n.RetryCount, &n.CreatedAt); err != nil {
			return nil, 0, err
		}
		if templateCode.Valid {
			n.TemplateCode = &templateCode.String
		}
		out = append(out, n)
	}
	return out, total, rows.Err()
}

func (r *PlatformRepository) ListAdminNotifications(ctx context.Context, channel, status string, limit, offset int) ([]Notification, int, error) {
	var total int
	err := r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM platform.notification
		WHERE ($1 = '' OR channel = $1) AND ($2 = '' OR status = $2)
	`, channel, status).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, code, template_id,
		       (SELECT code FROM platform.notification_template WHERE id = template_id),
		       recipient_type, recipient_id, channel, recipient_address, subject, body,
		       status, provider_message_id, sent_at, delivered_at, failed_at, error_message, retry_count, created_at
		FROM platform.notification
		WHERE ($1 = '' OR channel = $1) AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC, id DESC
		LIMIT $3 OFFSET $4
	`, channel, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Notification
	for rows.Next() {
		var n Notification
		var templateCode pgtype.Text
		if err := rows.Scan(&n.ID, &n.Code, &n.TemplateID, &templateCode, &n.RecipientType, &n.RecipientID,
			&n.Channel, &n.RecipientAddress, &n.Subject, &n.Body, &n.Status, &n.ProviderMessageID,
			&n.SentAt, &n.DeliveredAt, &n.FailedAt, &n.ErrorMessage, &n.RetryCount, &n.CreatedAt); err != nil {
			return nil, 0, err
		}
		if templateCode.Valid {
			n.TemplateCode = &templateCode.String
		}
		out = append(out, n)
	}
	return out, total, rows.Err()
}

func (r *PlatformRepository) GetNotificationByCode(ctx context.Context, code string) (*Notification, error) {
	var n Notification
	var templateCode pgtype.Text
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, code, template_id,
		       (SELECT code FROM platform.notification_template WHERE id = template_id),
		       recipient_type, recipient_id, channel, recipient_address, subject, body,
		       status, provider_message_id, sent_at, delivered_at, failed_at, error_message, retry_count, created_at
		FROM platform.notification
		WHERE code = $1
	`, code).Scan(&n.ID, &n.Code, &n.TemplateID, &templateCode, &n.RecipientType, &n.RecipientID,
		&n.Channel, &n.RecipientAddress, &n.Subject, &n.Body, &n.Status, &n.ProviderMessageID,
		&n.SentAt, &n.DeliveredAt, &n.FailedAt, &n.ErrorMessage, &n.RetryCount, &n.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if templateCode.Valid {
		n.TemplateCode = &templateCode.String
	}
	return &n, nil
}

type NotificationEvent struct {
	ID              int64
	NotificationID  int64
	EventType       string
	ProviderEventID *string
	Metadata        *string
	CreatedAt       time.Time
}

func (r *PlatformRepository) CreateNotificationEvent(ctx context.Context, notificationID int64, eventType string, providerEventID *string, metadata *string) error {
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO platform.notification_event (notification_id, event_type, provider_event_id, metadata)
		VALUES ($1, $2, $3, $4)
	`, notificationID, eventType, providerEventID, metadata)
	return err
}

func (r *PlatformRepository) ListNotificationEvents(ctx context.Context, notificationID int64) ([]NotificationEvent, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, notification_id, event_type, provider_event_id, metadata, created_at
		FROM platform.notification_event
		WHERE notification_id = $1
		ORDER BY created_at
	`, notificationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []NotificationEvent
	for rows.Next() {
		var e NotificationEvent
		if err := rows.Scan(&e.ID, &e.NotificationID, &e.EventType, &e.ProviderEventID, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *PlatformRepository) GetBuyerEmail(ctx context.Context, buyerID int64) (string, error) {
	var email string
	err := r.db.Pool.QueryRow(ctx, `
		SELECT contact_email FROM identity.buyer_profile WHERE id = $1
	`, buyerID).Scan(&email)
	if err == pgx.ErrNoRows {
		return "", database.ErrNotFound
	}
	return email, err
}

func (r *PlatformRepository) GetUserEmail(ctx context.Context, userID int64) (string, error) {
	var email string
	err := r.db.Pool.QueryRow(ctx, `
		SELECT email FROM identity."user" WHERE id = $1
	`, userID).Scan(&email)
	if err == pgx.ErrNoRows {
		return "", database.ErrNotFound
	}
	return email, err
}

type FeatureFlag struct {
	ID          int64
	Key         string
	Name        string
	Description *string
	Enabled     bool
	UpdatedBy   *int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (r *PlatformRepository) ListFeatureFlags(ctx context.Context, enabled *bool, limit, offset int) ([]FeatureFlag, int, error) {
	var total int
	countQuery := `SELECT COUNT(*)::int FROM platform.feature_flag`
	listQuery := `SELECT id, key, name, description, enabled, updated_by, created_at, updated_at FROM platform.feature_flag`
	var args []interface{}
	argIdx := 1

	if enabled != nil {
		countQuery += fmt.Sprintf(` WHERE enabled = $%d`, argIdx)
		listQuery += fmt.Sprintf(` WHERE enabled = $%d`, argIdx)
		args = append(args, *enabled)
		argIdx++
	}

	err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	listQuery += fmt.Sprintf(` ORDER BY key LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []FeatureFlag
	for rows.Next() {
		var f FeatureFlag
		var desc pgtype.Text
		var updatedBy pgtype.Int8
		if err := rows.Scan(&f.ID, &f.Key, &f.Name, &desc, &f.Enabled, &updatedBy, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if desc.Valid {
			f.Description = &desc.String
		}
		if updatedBy.Valid {
			f.UpdatedBy = &updatedBy.Int64
		}
		out = append(out, f)
	}
	return out, total, rows.Err()
}

func (r *PlatformRepository) GetFeatureFlagByKey(ctx context.Context, key string) (*FeatureFlag, error) {
	var f FeatureFlag
	var desc pgtype.Text
	var updatedBy pgtype.Int8
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, key, name, description, enabled, updated_by, created_at, updated_at
		FROM platform.feature_flag WHERE key = $1
	`, key).Scan(&f.ID, &f.Key, &f.Name, &desc, &f.Enabled, &updatedBy, &f.CreatedAt, &f.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if desc.Valid {
		f.Description = &desc.String
	}
	if updatedBy.Valid {
		f.UpdatedBy = &updatedBy.Int64
	}
	return &f, nil
}

func (r *PlatformRepository) UpsertFeatureFlag(ctx context.Context, key, name string, description *string, enabled bool, updatedBy *int64) (*FeatureFlag, error) {
	var f FeatureFlag
	var desc pgtype.Text
	var updatedByCol pgtype.Int8
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO platform.feature_flag (key, name, description, enabled, updated_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (key) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			enabled = EXCLUDED.enabled,
			updated_by = EXCLUDED.updated_by,
			updated_at = NOW()
		RETURNING id, key, name, description, enabled, updated_by, created_at, updated_at
	`, key, name, description, enabled, updatedBy).Scan(
		&f.ID, &f.Key, &f.Name, &desc, &f.Enabled, &updatedByCol, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if desc.Valid {
		f.Description = &desc.String
	}
	if updatedByCol.Valid {
		f.UpdatedBy = &updatedByCol.Int64
	}
	return &f, nil
}

type AuditLog struct {
	ID           int64
	ActorID      *int64
	ActorType    string
	Action       string
	ResourceType string
	ResourceID   *int64
	ResourceCode *string
	Metadata     *string
	IPAddress    *string
	UserAgent    *string
	CreatedAt    time.Time
}

func (r *PlatformRepository) ListAuditLog(ctx context.Context, actorType, action, resourceType string, actorID, resourceID *int64, limit, offset int) ([]AuditLog, int, error) {
	where := "1=1"
	var args []interface{}
	argIdx := 1

	if actorType != "" {
		where += fmt.Sprintf(` AND actor_type = $%d`, argIdx)
		args = append(args, actorType)
		argIdx++
	}
	if action != "" {
		where += fmt.Sprintf(` AND action = $%d`, argIdx)
		args = append(args, action)
		argIdx++
	}
	if resourceType != "" {
		where += fmt.Sprintf(` AND resource_type = $%d`, argIdx)
		args = append(args, resourceType)
		argIdx++
	}
	if actorID != nil {
		where += fmt.Sprintf(` AND actor_id = $%d`, argIdx)
		args = append(args, *actorID)
		argIdx++
	}
	if resourceID != nil {
		where += fmt.Sprintf(` AND resource_id = $%d`, argIdx)
		args = append(args, *resourceID)
		argIdx++
	}

	var total int
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM platform.audit_log WHERE `+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `SELECT id, actor_id, actor_type, action, resource_type, resource_id, resource_code, metadata, ip_address, user_agent, created_at FROM platform.audit_log WHERE ` + where
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []AuditLog
	for rows.Next() {
		var a AuditLog
		var actorIDCol, resourceIDCol pgtype.Int8
		var resourceCode, metadata, ipAddr, ua pgtype.Text
		if err := rows.Scan(&a.ID, &actorIDCol, &a.ActorType, &a.Action, &a.ResourceType, &resourceIDCol, &resourceCode, &metadata, &ipAddr, &ua, &a.CreatedAt); err != nil {
			return nil, 0, err
		}
		if actorIDCol.Valid {
			a.ActorID = &actorIDCol.Int64
		}
		if resourceIDCol.Valid {
			a.ResourceID = &resourceIDCol.Int64
		}
		if resourceCode.Valid {
			a.ResourceCode = &resourceCode.String
		}
		if metadata.Valid {
			a.Metadata = &metadata.String
		}
		if ipAddr.Valid {
			a.IPAddress = &ipAddr.String
		}
		if ua.Valid {
			a.UserAgent = &ua.String
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}

type IntegrationTraffic struct {
	ID              int64
	Direction       string
	Provider        string
	Endpoint        string
	Method          *string
	StatusCode      *int
	PayloadHash     *string
	RequestHeaders  *string
	ResponseHeaders *string
	ErrorMessage    *string
	DurationMS      *int
	CreatedAt       time.Time
}

func (r *PlatformRepository) ListIntegrationTraffic(ctx context.Context, direction, provider string, limit, offset int) ([]IntegrationTraffic, int, error) {
	where := "1=1"
	var args []interface{}
	argIdx := 1

	if direction != "" {
		where += fmt.Sprintf(` AND direction = $%d`, argIdx)
		args = append(args, direction)
		argIdx++
	}
	if provider != "" {
		where += fmt.Sprintf(` AND provider = $%d`, argIdx)
		args = append(args, provider)
		argIdx++
	}

	var total int
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM platform.integration_traffic WHERE `+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `SELECT id, direction, provider, endpoint, method, status_code, payload_hash, request_headers, response_headers, error_message, duration_ms, created_at FROM platform.integration_traffic WHERE ` + where
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []IntegrationTraffic
	for rows.Next() {
		var t IntegrationTraffic
		var method, payloadHash, reqHeaders, respHeaders, errMsg pgtype.Text
		var statusCode pgtype.Int8
		var durationMS pgtype.Int4
		if err := rows.Scan(&t.ID, &t.Direction, &t.Provider, &t.Endpoint, &method, &statusCode, &payloadHash, &reqHeaders, &respHeaders, &errMsg, &durationMS, &t.CreatedAt); err != nil {
			return nil, 0, err
		}
		if method.Valid {
			t.Method = &method.String
		}
		if statusCode.Valid {
			sc := int(statusCode.Int64)
			t.StatusCode = &sc
		}
		if payloadHash.Valid {
			t.PayloadHash = &payloadHash.String
		}
		if reqHeaders.Valid {
			t.RequestHeaders = &reqHeaders.String
		}
		if respHeaders.Valid {
			t.ResponseHeaders = &respHeaders.String
		}
		if errMsg.Valid {
			t.ErrorMessage = &errMsg.String
		}
		if durationMS.Valid {
			d := int(durationMS.Int32)
			t.DurationMS = &d
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

type MediaUpload struct {
	ID          int64
	Code        string
	Filename    string
	ContentType string
	SizeBytes   *int64
	UploadURL   string
	DownloadURL *string
	Status      string
	ExpiresAt   time.Time
	CreatedBy   *int64
	CreatedAt   time.Time
}

func (r *PlatformRepository) CreateMediaUpload(ctx context.Context, code, filename, contentType, uploadURL string, expiresAt time.Time, createdBy *int64) (*MediaUpload, error) {
	var m MediaUpload
	var downloadURL pgtype.Text
	var createdByCol pgtype.Int8
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO platform.media_upload (code, filename, content_type, upload_url, expires_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, code, filename, content_type, size_bytes, upload_url, download_url, status, expires_at, created_by, created_at
	`, code, filename, contentType, uploadURL, expiresAt, createdBy).Scan(
		&m.ID, &m.Code, &m.Filename, &m.ContentType, &m.SizeBytes, &m.UploadURL, &downloadURL, &m.Status, &m.ExpiresAt, &createdByCol, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	if downloadURL.Valid {
		m.DownloadURL = &downloadURL.String
	}
	if createdByCol.Valid {
		m.CreatedBy = &createdByCol.Int64
	}
	return &m, nil
}

func (r *PlatformRepository) GetMediaUploadByCode(ctx context.Context, code string) (*MediaUpload, error) {
	var m MediaUpload
	var downloadURL pgtype.Text
	var createdByCol pgtype.Int8
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, code, filename, content_type, size_bytes, upload_url, download_url, status, expires_at, created_by, created_at
		FROM platform.media_upload WHERE code = $1
	`, code).Scan(&m.ID, &m.Code, &m.Filename, &m.ContentType, &m.SizeBytes, &m.UploadURL, &downloadURL, &m.Status, &m.ExpiresAt, &createdByCol, &m.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if downloadURL.Valid {
		m.DownloadURL = &downloadURL.String
	}
	if createdByCol.Valid {
		m.CreatedBy = &createdByCol.Int64
	}
	return &m, nil
}

type ReferenceRegion struct {
	ID        int64
	Code      string
	Name      string
	Country   string
	IsActive  bool
	CreatedAt time.Time
}

func (r *PlatformRepository) ListActiveRegions(ctx context.Context) ([]ReferenceRegion, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, code, name, country, is_active, created_at
		FROM platform.reference_region
		WHERE is_active = TRUE
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ReferenceRegion
	for rows.Next() {
		var reg ReferenceRegion
		if err := rows.Scan(&reg.ID, &reg.Code, &reg.Name, &reg.Country, &reg.IsActive, &reg.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, reg)
	}
	return out, rows.Err()
}
