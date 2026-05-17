package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"confirmation-review-service/internal/model"
	"confirmation-review-service/internal/repository"
	"confirmation-review-service/internal/service"

	"github.com/gin-gonic/gin"
)

type CaseHandler struct {
	svc *service.CaseService
}

func NewCaseHandler(svc *service.CaseService) *CaseHandler {
	return &CaseHandler{svc: svc}
}

func (h *CaseHandler) Create(c *gin.Context) {
	var input repository.InsertCaseInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Kind == "" {
		input.Kind = "actionable"
	}

	if input.FlowSource != "mañana" && input.FlowSource != "citas" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "flow_source debe ser 'mañana' o 'citas'"})
		return
	}

	if input.IdempotencyKey == "" {
		input.IdempotencyKey = input.CitaID + "_" + input.FlowSource
		if input.Kind != "" {
			input.IdempotencyKey += "_" + input.Kind
		}
	}

	created, err := h.svc.Create(input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_ = repository.InsertAuditLog(created.ID, "created", nil, json.RawMessage(fmt.Sprintf(`{"source":"n8n","flow":"%s"}`, created.FlowSource)))

	c.JSON(http.StatusCreated, gin.H{"success": true, "item": created})
}

func (h *CaseHandler) ListPending(c *gin.Context) {
	kind := c.DefaultQuery("kind", "actionable")
	flowSource := c.DefaultQuery("flow_source", "")

	switch kind {
	case "informative":
		items, err := repository.ListInformative()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if items == nil {
			items = []model.ConfirmationCase{}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "items": items, "kind": "informative"})

	default:
		items, err := repository.ListPending(flowSource)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if items == nil {
			items = []model.ConfirmationCase{}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "items": items, "kind": "actionable"})
	}
}

func (h *CaseHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	cc, err := repository.GetCaseByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cc == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "caso no encontrado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "item": cc})
}

func (h *CaseHandler) Approve(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	email := c.GetString("user_email")
	updated, err := h.svc.Approve(id, email)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "item": updated})
}

func (h *CaseHandler) Skip(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	email := c.GetString("user_email")
	updated, err := h.svc.Skip(id, email)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "item": updated})
}

func (h *CaseHandler) Cancel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	email := c.GetString("user_email")
	updated, err := h.svc.Cancel(id, email)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "item": updated})
}

func (h *CaseHandler) RefreshChat(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	var body struct {
		ChatContext json.RawMessage `json:"chat_context" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.RefreshChat(id, body.ChatContext); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	email := c.GetString("user_email")
	_ = repository.InsertAuditLog(id, "chat_refreshed", &email, json.RawMessage(`{"action":"chat_refreshed"}`))

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *CaseHandler) CountPending(c *gin.Context) {
	flowSource := c.DefaultQuery("flow_source", "")
	count, err := repository.CountPending(flowSource)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "count": count})
}

func (h *CaseHandler) ExpireOld(c *gin.Context) {
	maxAgeDaysStr := c.DefaultQuery("max_age_days", "1")
	maxAgeDays, err := strconv.Atoi(maxAgeDaysStr)
	if err != nil {
		maxAgeDays = 1
	}

	if maxAgeDays < 1 {
		maxAgeDays = 1
	}

	count, err := h.svc.ExpireOld(maxAgeDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "expired": count})
}

type notifyInput struct {
	FlowSource  string    `json:"flow_source" binding:"required"`
	PendingCount *int     `json:"pending_count"`
	Timestamp   time.Time `json:"timestamp"`
}

func (h *CaseHandler) Notify(c *gin.Context) {
	var input notifyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pendingCount := 0
	if input.PendingCount != nil {
		pendingCount = *input.PendingCount
	} else {
		count, err := repository.CountPending(input.FlowSource)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		pendingCount = count
	}

	if err := service.NotifyWorkflowFinished(input.FlowSource, pendingCount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"pending_count": pendingCount,
		"notified":      pendingCount > 0 && service.HasSubscribers(),
	})
}
