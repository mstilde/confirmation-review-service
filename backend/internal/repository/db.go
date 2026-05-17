package repository

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Connect(databaseURL string) error {
	var err error
	Pool, err = pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return err
	}

	if err := Pool.Ping(context.Background()); err != nil {
		return err
	}

	log.Println("Conectado a PostgreSQL")
	return nil
}

func Close() {
	if Pool != nil {
		Pool.Close()
	}
}
