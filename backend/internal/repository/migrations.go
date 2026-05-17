package repository

import (
	"context"
)

func RunMigrations() error {
	ctx := context.Background()

	migrations := []string{

		`CREATE TABLE IF NOT EXISTS app_users (
			id BIGSERIAL PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS confirmation_cases (
			id BIGSERIAL PRIMARY KEY,
			idempotency_key TEXT UNIQUE NOT NULL,
			cita_id TEXT NOT NULL,
			contact_name TEXT,
			appointment_at TIMESTAMPTZ,
			flow_source TEXT NOT NULL,
			ai_reason TEXT,
			chat_context JSONB DEFAULT '[]'::jsonb,
			suggested_message TEXT,
			account_id TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			resolved_by TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			resolved_at TIMESTAMPTZ,
			expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '1 day'),
			kind TEXT NOT NULL DEFAULT 'actionable',
			skip_reason TEXT
		)`,

		`CREATE INDEX IF NOT EXISTS idx_cc_status ON confirmation_cases(status, kind, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_cc_expires ON confirmation_cases(expires_at) WHERE status = 'pending'`,

		`CREATE TABLE IF NOT EXISTS case_audit_log (
			id BIGSERIAL PRIMARY KEY,
			case_id BIGINT REFERENCES confirmation_cases(id) ON DELETE CASCADE,
			action TEXT NOT NULL,
			performed_by TEXT,
			details JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS push_subscriptions (
			id BIGSERIAL PRIMARY KEY,
			user_email TEXT NOT NULL,
			endpoint TEXT NOT NULL UNIQUE,
			p256dh TEXT NOT NULL,
			auth TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
	}

	for _, m := range migrations {
		if _, err := Pool.Exec(ctx, m); err != nil {
			return err
		}
	}

	return nil
}

func SavePushSubscription(userEmail, endpoint, p256dh, auth string) error {
	ctx := context.Background()
	_, err := Pool.Exec(ctx, `
		INSERT INTO push_subscriptions (user_email, endpoint, p256dh, auth)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (endpoint) DO UPDATE SET
			user_email = EXCLUDED.user_email,
			p256dh = EXCLUDED.p256dh,
			auth = EXCLUDED.auth
	`, userEmail, endpoint, p256dh, auth)
	return err
}

func GetAllPushSubscriptions() ([]struct {
	UserEmail string
	Endpoint  string
	P256DH    string
	Auth      string
}, error) {
	ctx := context.Background()
	rows, err := Pool.Query(ctx, `
		SELECT user_email, endpoint, p256dh, auth FROM push_subscriptions
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []struct {
		UserEmail string
		Endpoint  string
		P256DH    string
		Auth      string
	}
	for rows.Next() {
		var s struct {
			UserEmail string
			Endpoint  string
			P256DH    string
			Auth      string
		}
		if err := rows.Scan(&s.UserEmail, &s.Endpoint, &s.P256DH, &s.Auth); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}
