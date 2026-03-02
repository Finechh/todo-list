package mocks

import (
	"context"
	"todo_list/internal/models"

	"github.com/stretchr/testify/mock"
)

type MockTodoRepository struct {
	mock.Mock
}

func (m *MockTodoRepository) CreateTodolist(todo models.Todo, ctx context.Context) error {
	return m.Called(todo, ctx).Error(0)
}

func (m *MockTodoRepository) GetTodolist(id string, ctx context.Context) (models.Todo, error) {
	args := m.Called(id, ctx)
	return args.Get(0).(models.Todo), args.Error(1)
}

func (m *MockTodoRepository) GetAllTodolist(ctx context.Context) ([]models.Todo, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Todo), args.Error(1)
}

func (m *MockTodoRepository) UpdateTodolist(todo models.Todo, ctx context.Context) error {
	return m.Called(todo, ctx).Error(0)
}

func (m *MockTodoRepository) DeleteTodolist(id string, ctx context.Context) error {
	return m.Called(id, ctx).Error(0)
}

func (m *MockTodoRepository) DeleteAllTodolist(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}
