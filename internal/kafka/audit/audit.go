package audit

import (
	"context"
	"encoding/json"
	"time"
	"todo_list/internal/metrics"
	"todo_list/internal/models"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
)

type TodoEventProducer interface {
	SendTodoEvent(ctx context.Context, eventType string, payload []byte) error
}

type KafkaTodoEvent struct {
	producer sarama.SyncProducer
	topic    string
}

func NewKafkaTodoEvent(p sarama.SyncProducer, topic string) *KafkaTodoEvent {
	return &KafkaTodoEvent{producer: p, topic: topic}
}

func (s *KafkaTodoEvent) SendTodoEvent(ctx context.Context, eventType string, payload []byte) error {
	start := time.Now()
	metrics.ObserveBackendDuration("kafka", "audit_publish", time.Since(start))
	event := &models.KafkaEvent{
		EventID:   uuid.NewString(),
		Type:      eventType,
		Timestamp: time.Now(),
		Version:   1,
		Source:    "todo_service",
		Payload:   payload,
	}

	bytes, err := json.Marshal(event)
	if err != nil {
		metrics.IncBackendError("kafka", "audit_publish")
		return err
	}
	msg := &sarama.ProducerMessage{
		Topic: s.topic,
		Value: sarama.ByteEncoder(bytes),
	}

	_, _, err = s.producer.SendMessage(msg)
	if err != nil {
		metrics.IncBackendError("kafka", "audit_publish")
		return err
	}
	metrics.IncBackendSuccess("kafka", "audit_publish")
	return nil
}
