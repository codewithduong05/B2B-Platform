package schema

import "time"

type NotificationTemplateResponse struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Channel   string    `json:"channel"`
	Subject   *string   `json:"subject,omitempty"`
	Body      string    `json:"body_template"`
	Variables []string  `json:"variables"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateTemplateRequest struct {
	Code      string   `json:"code"`
	Name      string   `json:"name"`
	Channel   string   `json:"channel"`
	Subject   *string  `json:"subject,omitempty"`
	Body      string   `json:"body_template"`
	Variables []string `json:"variables,omitempty"`
}

type UpdateTemplateRequest struct {
	Name      *string  `json:"name,omitempty"`
	Subject   *string  `json:"subject,omitempty"`
	Body      *string  `json:"body_template,omitempty"`
	Variables []string `json:"variables,omitempty"`
	Status    *string  `json:"status,omitempty"`
}

type NotificationResponse struct {
	ID                int64      `json:"id"`
	Code              string     `json:"code"`
	TemplateCode      *string    `json:"template_code,omitempty"`
	RecipientType     string     `json:"recipient_type"`
	RecipientID       int64      `json:"recipient_id"`
	Channel           string     `json:"channel"`
	RecipientAddress  string     `json:"recipient_address"`
	Subject           *string    `json:"subject,omitempty"`
	Body              string     `json:"body"`
	Status            string     `json:"status"`
	ProviderMessageID *string    `json:"provider_message_id,omitempty"`
	SentAt            *time.Time `json:"sent_at,omitempty"`
	DeliveredAt       *time.Time `json:"delivered_at,omitempty"`
	FailedAt          *time.Time `json:"failed_at,omitempty"`
	ErrorMessage      *string    `json:"error_message,omitempty"`
	RetryCount        int        `json:"retry_count"`
	CreatedAt         time.Time  `json:"created_at"`
}

type NotificationEventResponse struct {
	ID              int64     `json:"id"`
	NotificationID  int64     `json:"notification_id"`
	EventType       string    `json:"event_type"`
	ProviderEventID *string   `json:"provider_event_id,omitempty"`
	Metadata        *string   `json:"metadata,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type SendNotificationRequest struct {
	TemplateCode  string                 `json:"template_code"`
	RecipientType string                 `json:"recipient_type"`
	RecipientID   int64                  `json:"recipient_id"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

type BuyerNotificationResponse struct {
	ID        int64      `json:"id"`
	Code      string     `json:"code"`
	Subject   *string    `json:"subject,omitempty"`
	Body      string     `json:"body"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
}

type CreateMediaRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

type MediaResponse struct {
	Code        string    `json:"code"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	UploadURL   string    `json:"upload_url"`
	DownloadURL *string   `json:"download_url,omitempty"`
	Status      string    `json:"status"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type RegionResponse struct {
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Country   string    `json:"country"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type FeatureFlagResponse struct {
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Enabled     bool      `json:"enabled"`
	UpdatedBy   *int64    `json:"updated_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UpdateFeatureFlagRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Enabled     bool    `json:"enabled"`
}

type AuditLogResponse struct {
	ID           int64     `json:"id"`
	ActorID      *int64    `json:"actor_id,omitempty"`
	ActorType    string    `json:"actor_type"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   *int64    `json:"resource_id,omitempty"`
	ResourceCode *string   `json:"resource_code,omitempty"`
	Metadata     *string   `json:"metadata,omitempty"`
	IPAddress    *string   `json:"ip_address,omitempty"`
	UserAgent    *string   `json:"user_agent,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type IntegrationTrafficResponse struct {
	ID              int64     `json:"id"`
	Direction       string    `json:"direction"`
	Provider        string    `json:"provider"`
	Endpoint        string    `json:"endpoint"`
	Method          *string   `json:"method,omitempty"`
	StatusCode      *int      `json:"status_code,omitempty"`
	PayloadHash     *string   `json:"payload_hash,omitempty"`
	RequestHeaders  *string   `json:"request_headers,omitempty"`
	ResponseHeaders *string   `json:"response_headers,omitempty"`
	ErrorMessage    *string   `json:"error_message,omitempty"`
	DurationMS      *int      `json:"duration_ms,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}
