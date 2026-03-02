package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockKafkaProducer struct {
	mock.Mock
}

func (m *MockKafkaProducer) SendTodoEvent(ctx context.Context, eventType string, payload []byte) error {
	return m.Called(ctx, eventType, payload).Error(0)
}
