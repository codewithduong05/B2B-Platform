package schema

import (
	"time"
)

type PartnerResponse struct {
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreatePartnerRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug,omitempty"`
}

type ReferralResponse struct {
	Code      string    `json:"code"`
	PartnerID int64     `json:"partner_id"`
	IsActive  bool      `json:"is_active"`
	Uses      int       `json:"uses"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateReferralRequest struct {
	PartnerID int64  `json:"partner_id"`
	Code      string `json:"code,omitempty"`
}

type LeadResponse struct {
	Code         string    `json:"code"`
	ContactName  string    `json:"contact_name"`
	BusinessName string    `json:"business_name"`
	Email        string    `json:"email,omitempty"`
	Phone        string    `json:"phone,omitempty"`
	Message      string    `json:"message,omitempty"`
	Status       string    `json:"status"`
	PartnerID    *int64    `json:"partner_id,omitempty"`
	ReferralID   *int64    `json:"referral_id,omitempty"`
	AssignedTo   *int64    `json:"assigned_to,omitempty"`
	BuyerID      *int64    `json:"buyer_id,omitempty"`
	Notes        string    `json:"notes,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type SubmitLeadRequest struct {
	ContactName  string `json:"contact_name"`
	BusinessName string `json:"business_name"`
	Email        string `json:"email,omitempty"`
	Phone        string `json:"phone,omitempty"`
	Message      string `json:"message,omitempty"`
	PartnerRef   string `json:"partner_ref,omitempty"`
}

type AssignLeadRequest struct {
	Assignee int64 `json:"assignee"`
}

type UpdateLeadRequest struct {
	Status  *string `json:"status,omitempty"`
	Notes   *string `json:"notes,omitempty"`
	BuyerID *int64  `json:"buyer_id,omitempty"`
}

type TrackResponse struct {
	Partner PartnerResponse `json:"partner"`
}
