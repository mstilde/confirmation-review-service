package main

import (
	"fmt"
	"log"

	"confirmation-review-service/internal/config"
	"confirmation-review-service/internal/handler"
	"confirmation-review-service/internal/repository"
)

func main() {
	cfg := config.Load()

	if err := repository.Connect(cfg.DatabaseURL); err != nil {
		log.Fatalf("Error conectando a PostgreSQL: %v", err)
	}
	defer repository.Close()

	if err := repository.RunMigrations(); err != nil {
		log.Fatalf("Error corriendo migraciones: %v", err)
	}

	r := handler.SetupRouter(cfg)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Backend iniciado en http://localhost%s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Error iniciando servidor: %v", err)
	}
}
