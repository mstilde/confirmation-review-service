package repository

import (
	"context"
)

func FindUserByEmail(email string) (string, string, error) {
	ctx := context.Background()
	var foundEmail, passwordHash string
	err := Pool.QueryRow(ctx, `
		SELECT email, password_hash FROM app_users WHERE email = $1
	`, email).Scan(&foundEmail, &passwordHash)
	return foundEmail, passwordHash, err
}

func CreateUser(email, passwordHash string) error {
	ctx := context.Background()
	_, err := Pool.Exec(ctx, `
		INSERT INTO app_users (email, password_hash) VALUES ($1, $2)
		ON CONFLICT (email) DO UPDATE SET password_hash = EXCLUDED.password_hash
	`, email, passwordHash)
	return err
}
