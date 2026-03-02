package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

type MockCache struct {
	mock.Mock
}

func (m *MockCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return m.Called(ctx, key, value, ttl).Error(0)
}

func (m *MockCache) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockCache) Del(ctx context.Context, key string) error {
	return m.Called(ctx, key).Error(0)
}

func (m *MockCache) Close() error {
	return m.Called().Error(0)
}
