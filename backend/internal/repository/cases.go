package repository

import (
	"context"
	"encoding/json"
	"time"

	"confirmation-review-service/internal/model"

	"github.com/jackc/pgx/v5"
)

func CreateCase(c model.ConfirmationCase) (*model.ConfirmationCase, error) {
	ctx := context.Background()

	chatJSON := json.RawMessage(`[]`)
	if len(c.ChatContext) > 0 {
		chatJSON = c.ChatContext
	}

	var row model.ConfirmationCase
	err := Pool.QueryRow(ctx, `
		INSERT INTO confirmation_cases
			(idempotency_key, cita_id, chat_id, contact_name, appointment_at, flow_source,
			 ai_reason, chat_context, suggested_message, account_id, kind, skip_reason, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (idempotency_key) DO UPDATE SET
			chat_id = EXCLUDED.chat_id,
			contact_name = EXCLUDED.contact_name,
			appointment_at = EXCLUDED.appointment_at,
			ai_reason = EXCLUDED.ai_reason,
			chat_context = EXCLUDED.chat_context,
			suggested_message = EXCLUDED.suggested_message,
			account_id = EXCLUDED.account_id,
			skip_reason = EXCLUDED.skip_reason,
			status = 'pending',
			resolved_at = NULL,
			expires_at = EXCLUDED.expires_at,
			created_at = NOW()
		RETURNING id, idempotency_key, cita_id, chat_id, contact_name, appointment_at, flow_source,
				  ai_reason, chat_context, suggested_message, account_id, status,
				  resolved_by, created_at, resolved_at, expires_at, kind, skip_reason
	`, c.IdempotencyKey, c.CitaID, c.ChatID, c.ContactName, c.AppointmentAt, c.FlowSource,
		c.AIReason, chatJSON, c.SuggestedMessage, c.AccountID, c.Kind, c.SkipReason, c.ExpiresAt).
		Scan(&row.ID, &row.IdempotencyKey, &row.CitaID, &row.ChatID, &row.ContactName, &row.AppointmentAt,
			&row.FlowSource, &row.AIReason, &row.ChatContext, &row.SuggestedMessage,
			&row.AccountID, &row.Status, &row.ResolvedBy, &row.CreatedAt, &row.ResolvedAt,
			&row.ExpiresAt, &row.Kind, &row.SkipReason)

	if err != nil {
		return nil, err
	}
	return &row, nil
}

func ListPending(flowSource string) ([]model.ConfirmationCase, error) {
	ctx := context.Background()

	var rows pgx.Rows
	var err error

	if flowSource == "" {
		rows, err = Pool.Query(ctx, `
			SELECT id, idempotency_key, cita_id, chat_id, contact_name, appointment_at, flow_source,
				   ai_reason, chat_context, suggested_message, account_id, status,
				   resolved_by, created_at, resolved_at, expires_at, kind, skip_reason
			FROM confirmation_cases
			WHERE status = 'pending' AND kind = 'actionable'
			ORDER BY appointment_at ASC NULLS LAST, created_at ASC
		`)
	} else {
		rows, err = Pool.Query(ctx, `
			SELECT id, idempotency_key, cita_id, chat_id, contact_name, appointment_at, flow_source,
				   ai_reason, chat_context, suggested_message, account_id, status,
				   resolved_by, created_at, resolved_at, expires_at, kind, skip_reason
			FROM confirmation_cases
			WHERE status = 'pending' AND kind = 'actionable' AND flow_source = $1
			ORDER BY appointment_at ASC NULLS LAST, created_at ASC
		`, flowSource)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCases(rows)
}

func ListInformative() ([]model.ConfirmationCase, error) {
	ctx := context.Background()

	rows, err := Pool.Query(ctx, `
		SELECT id, idempotency_key, cita_id, chat_id, contact_name, appointment_at, flow_source,
			   ai_reason, chat_context, suggested_message, account_id, status,
			   resolved_by, created_at, resolved_at, expires_at, kind, skip_reason
		FROM confirmation_cases
		WHERE kind = 'informative'
		  AND created_at >= date_trunc('day', NOW())
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCases(rows)
}

func GetCaseByID(id int64) (*model.ConfirmationCase, error) {
	ctx := context.Background()

	var row model.ConfirmationCase
	err := Pool.QueryRow(ctx, `
		SELECT id, idempotency_key, cita_id, chat_id, contact_name, appointment_at, flow_source,
			   ai_reason, chat_context, suggested_message, account_id, status,
			   resolved_by, created_at, resolved_at, expires_at, kind, skip_reason
		FROM confirmation_cases
		WHERE id = $1
	`, id).
		Scan(&row.ID, &row.IdempotencyKey, &row.CitaID, &row.ChatID, &row.ContactName, &row.AppointmentAt,
			&row.FlowSource, &row.AIReason, &row.ChatContext, &row.SuggestedMessage,
			&row.AccountID, &row.Status, &row.ResolvedBy, &row.CreatedAt, &row.ResolvedAt,
			&row.ExpiresAt, &row.Kind, &row.SkipReason)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func UpdateCaseStatus(id int64, status model.CaseStatus, resolvedBy string) (*model.ConfirmationCase, error) {
	ctx := context.Background()

	var row model.ConfirmationCase
	err := Pool.QueryRow(ctx, `
		UPDATE confirmation_cases
		SET status = $2, resolved_at = NOW(), resolved_by = $3
		WHERE id = $1 AND status = 'pending'
		RETURNING id, idempotency_key, cita_id, chat_id, contact_name, appointment_at, flow_source,
				  ai_reason, chat_context, suggested_message, account_id, status,
				  resolved_by, created_at, resolved_at, expires_at, kind, skip_reason
	`, id, status, resolvedBy).
		Scan(&row.ID, &row.IdempotencyKey, &row.CitaID, &row.ChatID, &row.ContactName, &row.AppointmentAt,
			&row.FlowSource, &row.AIReason, &row.ChatContext, &row.SuggestedMessage,
			&row.AccountID, &row.Status, &row.ResolvedBy, &row.CreatedAt, &row.ResolvedAt,
			&row.ExpiresAt, &row.Kind, &row.SkipReason)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func RefreshChatContext(id int64, chatContext json.RawMessage) error {
	ctx := context.Background()
	_, err := Pool.Exec(ctx, `
		UPDATE confirmation_cases
		SET chat_context = $2
		WHERE id = $1
	`, id, chatContext)
	return err
}

func ExpireOldCases(maxAgeDays int) (int64, error) {
	ctx := context.Background()
	tag, err := Pool.Exec(ctx, `
		UPDATE confirmation_cases
		SET status = 'expired', resolved_at = NOW()
		WHERE status = 'pending' AND created_at < NOW() - ($1 || ' days')::INTERVAL
	`, maxAgeDays)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func CountPending(flowSource string) (int, error) {
	ctx := context.Background()
	var count int
	var err error

	if flowSource == "" {
		err = Pool.QueryRow(ctx, `
			SELECT COUNT(*)::int FROM confirmation_cases
			WHERE status = 'pending' AND kind = 'actionable'
		`).Scan(&count)
	} else {
		err = Pool.QueryRow(ctx, `
			SELECT COUNT(*)::int FROM confirmation_cases
			WHERE status = 'pending' AND kind = 'actionable' AND flow_source = $1
		`).Scan(&count)
	}
	return count, err
}

func InsertAuditLog(caseID int64, action string, performedBy *string, details json.RawMessage) error {
	ctx := context.Background()
	detailsJSON := json.RawMessage(`{}`)
	if len(details) > 0 {
		detailsJSON = details
	}
	_, err := Pool.Exec(ctx, `
		INSERT INTO case_audit_log (case_id, action, performed_by, details)
		VALUES ($1, $2, $3, $4)
	`, caseID, action, performedBy, detailsJSON)
	return err
}

func scanCases(rows pgx.Rows) ([]model.ConfirmationCase, error) {
	var cases []model.ConfirmationCase
	for rows.Next() {
		var c model.ConfirmationCase
		err := rows.Scan(&c.ID, &c.IdempotencyKey, &c.CitaID, &c.ChatID, &c.ContactName, &c.AppointmentAt,
			&c.FlowSource, &c.AIReason, &c.ChatContext, &c.SuggestedMessage,
			&c.AccountID, &c.Status, &c.ResolvedBy, &c.CreatedAt, &c.ResolvedAt,
			&c.ExpiresAt, &c.Kind, &c.SkipReason)
		if err != nil {
			return nil, err
		}
		cases = append(cases, c)
	}
	return cases, rows.Err()
}

type InsertCaseInput struct {
	IdempotencyKey   string           `json:"idempotency_key"`
	CitaID           string           `json:"cita_id" binding:"required"`
	ChatID           *string          `json:"chat_id"`
	ContactName      *string          `json:"contact_name"`
	AppointmentAt    *time.Time       `json:"appointment_at"`
	FlowSource       string           `json:"flow_source" binding:"required"`
	AIReason         *string          `json:"reason"`
	ChatContext      json.RawMessage  `json:"chat_context"`
	SuggestedMessage *string          `json:"suggested_message"`
	AccountID        *string          `json:"account_id"`
	Kind             string           `json:"kind"`
	SkipReason       *string          `json:"skip_reason"`
	ExpiresAt        *time.Time       `json:"expires_at"`
}
