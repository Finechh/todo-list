package models

import (
	"time"
)

type Todo struct {
	ID          string     `gorm:"primaryKey" json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Completed   bool       `json:"completed" gorm:"default:false"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at" gorm:"autoUpdateTime:false"`
}

type CreateTodoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateTodoRequest struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type KafkaEvent struct {
	EventID   string    `json:"event_id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Version   int       `json:"version"`
	Source    string    `json:"source"`
	Payload   []byte    `json:"payload"`
}
