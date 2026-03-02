package handlerAudit

import (
	"context"
	"encoding/json"
	"log"
	auditRepo "todo_list/internal/kafka/audit/repository"
	"todo_list/internal/models"
)

type Handler struct {
	repo *auditRepo.AuditRepo
}

func NewHandler(repo *auditRepo.AuditRepo) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Handle(ctx context.Context, msg []byte) error {
	var event models.KafkaEvent

	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}

	if err := h.repo.Create(ctx, event); err != nil {
		log.Printf("audit create error: %v", err)
	}
	return nil
}
