package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"confirmation-review-service/internal/model"
	"confirmation-review-service/internal/repository"
)

type CaseService struct {
	N8NWebhookURL string
}

func NewCaseService(n8nWebhookURL string) *CaseService {
	return &CaseService{N8NWebhookURL: n8nWebhookURL}
}

func (s *CaseService) Create(input repository.InsertCaseInput) (*model.ConfirmationCase, error) {
	expiresAt := time.Now().Add(24 * time.Hour)
	if input.ExpiresAt != nil {
		expiresAt = *input.ExpiresAt
	}

	chatJSON := json.RawMessage(`[]`)
	if len(input.ChatContext) > 0 {
		chatJSON = input.ChatContext
	}

	c := model.ConfirmationCase{
		IdempotencyKey:   input.IdempotencyKey,
		CitaID:           input.CitaID,
		ContactName:      input.ContactName,
		AppointmentAt:    input.AppointmentAt,
		FlowSource:       input.FlowSource,
		AIReason:         input.AIReason,
		ChatContext:      chatJSON,
		SuggestedMessage: input.SuggestedMessage,
		AccountID:        input.AccountID,
		Kind:             input.Kind,
		SkipReason:       input.SkipReason,
		ExpiresAt:        expiresAt,
	}

	created, err := repository.CreateCase(c)
	if err != nil {
		return nil, err
	}

	details := json.RawMessage(fmt.Sprintf(`{"action":"created","idempotency_key":"%s"}`, created.IdempotencyKey))
	_ = repository.InsertAuditLog(created.ID, "created", nil, details)

	return created, nil
}

func (s *CaseService) Approve(caseID int64, userEmail string) (*model.ConfirmationCase, error) {
	c, err := repository.GetCaseByID(caseID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("caso no encontrado")
	}
	if !model.CanTransition(c.Status, model.StatusApproved) {
		return nil, fmt.Errorf("no se puede aprobar un caso en estado '%s'", c.Status)
	}

	if s.N8NWebhookURL != "" {
		if err := s.dispatchToN8N(c, "send"); err != nil {
			details := json.RawMessage(fmt.Sprintf(`{"error":"%s"}`, err.Error()))
			_ = repository.InsertAuditLog(c.ID, "webhook_failed", &userEmail, details)
			return nil, fmt.Errorf("error al contactar n8n: %w", err)
		}
	}

	updated, err := repository.UpdateCaseStatus(caseID, model.StatusApproved, userEmail)
	if err != nil {
		return nil, err
	}

	details := json.RawMessage(`{"action":"approved"}`)
	_ = repository.InsertAuditLog(caseID, "approved", &userEmail, details)

	return updated, nil
}

func (s *CaseService) Skip(caseID int64, userEmail string) (*model.ConfirmationCase, error) {
	c, err := repository.GetCaseByID(caseID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("caso no encontrado")
	}
	if !model.CanTransition(c.Status, model.StatusSkipped) {
		return nil, fmt.Errorf("no se puede skipear un caso en estado '%s'", c.Status)
	}

	updated, err := repository.UpdateCaseStatus(caseID, model.StatusSkipped, userEmail)
	if err != nil {
		return nil, err
	}

	details := json.RawMessage(`{"action":"skipped"}`)
	_ = repository.InsertAuditLog(caseID, "skipped", &userEmail, details)

	return updated, nil
}

func (s *CaseService) Cancel(caseID int64, userEmail string) (*model.ConfirmationCase, error) {
	c, err := repository.GetCaseByID(caseID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("caso no encontrado")
	}
	if !model.CanTransition(c.Status, model.StatusCancelled) {
		return nil, fmt.Errorf("no se puede cancelar un caso en estado '%s'", c.Status)
	}

	if s.N8NWebhookURL != "" {
		if err := s.dispatchToN8N(c, "cancel"); err != nil {
			details := json.RawMessage(fmt.Sprintf(`{"error":"%s"}`, err.Error()))
			_ = repository.InsertAuditLog(c.ID, "webhook_failed", &userEmail, details)
			return nil, fmt.Errorf("error al contactar n8n: %w", err)
		}
	}

	updated, err := repository.UpdateCaseStatus(caseID, model.StatusCancelled, userEmail)
	if err != nil {
		return nil, err
	}

	details := json.RawMessage(`{"action":"cancelled"}`)
	_ = repository.InsertAuditLog(caseID, "cancelled", &userEmail, details)

	return updated, nil
}

func (s *CaseService) RefreshChat(caseID int64, chatContext json.RawMessage) error {
	return repository.RefreshChatContext(caseID, chatContext)
}

func (s *CaseService) ExpireOld(maxAgeDays int) (int64, error) {
	count, err := repository.ExpireOldCases(maxAgeDays)
	if err != nil {
		return 0, err
	}
	if count > 0 {
		log.Printf("[expire] %d casos marcados como expirados", count)
	}
	return count, nil
}

func (s *CaseService) dispatchToN8N(c *model.ConfirmationCase, action string) error {
	payload := map[string]interface{}{
		"action":           action,
		"pending_id":       c.ID,
		"cita_id":          c.CitaID,
		"contact_name":     c.ContactName,
		"flow_source":      c.FlowSource,
		"account_id":       c.AccountID,
		"message":          nil,
		"appointment_at":   c.AppointmentAt,
		"suggested_message": c.SuggestedMessage,
	}

	if action == "send" && c.SuggestedMessage != nil {
		payload["message"] = *c.SuggestedMessage
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(s.N8NWebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("n8n respondió HTTP %d", resp.StatusCode)
	}

	return nil
}
