package model

import (
	"encoding/json"
	"time"
)

type CaseStatus string

const (
	StatusPending   CaseStatus = "pending"
	StatusApproved  CaseStatus = "approved"
	StatusSkipped   CaseStatus = "skipped"
	StatusCancelled CaseStatus = "cancelled"
	StatusExpired   CaseStatus = "expired"
)

var ValidTransitions = map[CaseStatus][]CaseStatus{
	StatusPending: {StatusApproved, StatusSkipped, StatusCancelled, StatusExpired},
}

func CanTransition(from, to CaseStatus) bool {
	targets, ok := ValidTransitions[from]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

type ConfirmationCase struct {
	ID               int64            `json:"id"`
	IdempotencyKey   string           `json:"idempotency_key"`
	CitaID           string           `json:"cita_id"`
	ContactName      *string          `json:"contact_name"`
	AppointmentAt    *time.Time       `json:"appointment_at"`
	FlowSource       string           `json:"flow_source"`
	AIReason         *string          `json:"ai_reason"`
	ChatContext      json.RawMessage  `json:"chat_context"`
	SuggestedMessage *string          `json:"suggested_message"`
	AccountID        *string          `json:"account_id"`
	Status           CaseStatus       `json:"status"`
	ResolvedBy       *string          `json:"resolved_by"`
	CreatedAt        time.Time        `json:"created_at"`
	ResolvedAt       *time.Time       `json:"resolved_at"`
	ExpiresAt        time.Time        `json:"expires_at"`
	Kind             string           `json:"kind"`
	SkipReason       *string          `json:"skip_reason"`
}

type AuditLog struct {
	ID          int64            `json:"id"`
	CaseID      int64            `json:"case_id"`
	Action      string           `json:"action"`
	PerformedBy *string          `json:"performed_by"`
	Details     json.RawMessage  `json:"details"`
	CreatedAt   time.Time        `json:"created_at"`
}
