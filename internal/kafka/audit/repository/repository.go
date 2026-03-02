package auditRepo

import (
	"context"
	"todo_list/internal/models"

	"gorm.io/gorm"
)

type AuditRepository interface {
	Create(ctx context.Context, record models.KafkaEvent) error
}

type AuditRepo struct {
	db *gorm.DB
}

func NewAuditRepo(db *gorm.DB) *AuditRepo {
	return &AuditRepo{db: db}
}

func (r *AuditRepo) Create(ctx context.Context, event models.KafkaEvent) error {
	return r.db.WithContext(ctx).Create(&event).Error
}
