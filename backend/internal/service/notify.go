package service

import (
	"encoding/json"
	"fmt"
	"log"

	"confirmation-review-service/internal/repository"
)

func NotifyWorkflowFinished(flowSource string, pendingCount int) error {
	if pendingCount == 0 {
		log.Printf("[notify] workflow '%s' terminó con 0 casos pendientes — no se envía push", flowSource)
		return nil
	}

	subs, err := repository.GetAllPushSubscriptions()
	if err != nil {
		return fmt.Errorf("error obteniendo subscripciones push: %w", err)
	}

	if len(subs) == 0 {
		log.Printf("[notify] %d casos pendientes pero 0 dispositivos suscriptos — no se envía push", pendingCount)
		return nil
	}

	title := "Casos para revisar"
	body := fmt.Sprintf("El workflow '%s' terminó. %d caso(s) pendiente(s) de revisión.", flowSource, pendingCount)

	for _, sub := range subs {
		if err := sendWebPush(sub.Endpoint, sub.P256DH, sub.Auth, title, body); err != nil {
			log.Printf("[notify] error enviando push a %s: %v", sub.UserEmail, err)
		}
	}

	return nil
}

func sendWebPush(endpoint, p256dh, auth, title, body string) error {
	type pushMessage struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		Icon  string `json:"icon"`
		URL   string `json:"url"`
	}

	msg := pushMessage{
		Title: title,
		Body:  body,
		URL:   "/review",
	}

	msgJSON, _ := json.Marshal(msg)

	payload := map[string]interface{}{
		"subscription": map[string]interface{}{
			"endpoint": endpoint,
			"keys": map[string]string{
				"p256dh": p256dh,
				"auth":   auth,
			},
		},
		"message": msgJSON,
	}

	payloadBytes, _ := json.Marshal(payload)

	return sendPushToWorker(payloadBytes)
}

func sendPushToWorker(payload []byte) error {
	return nil
}

func HasSubscribers() bool {
	subs, err := repository.GetAllPushSubscriptions()
	if err != nil {
		return false
	}
	return len(subs) > 0
}
