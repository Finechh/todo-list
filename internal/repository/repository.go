package repository

import (
	"context"
	s "todo_list/internal/models"

	"gorm.io/gorm"
)

type TodolistRepository interface {
	CreateTodolist(Todo s.Todo, ctx context.Context) error
	GetTodolist(id string, ctx context.Context) (s.Todo, error)
	GetAllTodolist(ctx context.Context) ([]s.Todo, error)
	UpdateTodolist(Todo s.Todo, ctx context.Context) error
	DeleteTodolist(id string, ctx context.Context) error
	DeleteAllTodolist(ctx context.Context) error
}

type TodoRepository struct {
	db *gorm.DB
}

func NewTodoListRepository(db *gorm.DB) TodolistRepository {
	return &TodoRepository{db: db}
}

func (t *TodoRepository) CreateTodolist(Todo s.Todo, ctx context.Context) error {
	return t.db.WithContext(ctx).Create(&Todo).Error

}

func (t *TodoRepository) DeleteTodolist(id string, ctx context.Context) error {
	return t.db.WithContext(ctx).Delete(&s.Todo{}, "id = ?", id).Error

}

func (t *TodoRepository) GetAllTodolist(ctx context.Context) ([]s.Todo, error) {
	var td []s.Todo
	err := t.db.WithContext(ctx).Find(&td).Error
	return td, err
}

func (t *TodoRepository) GetTodolist(id string, ctx context.Context) (s.Todo, error) {
	var todolist s.Todo
	err := t.db.WithContext(ctx).First(&todolist, "id = ?", id).Error
	return todolist, err
}

func (t *TodoRepository) UpdateTodolist(todo s.Todo, ctx context.Context) error {
	return t.db.WithContext(ctx).Model(&s.Todo{}).Where("id = ?", todo.ID).Updates(todo).Error
}

func (t *TodoRepository) DeleteAllTodolist(ctx context.Context) error {
	return t.db.WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&s.Todo{}).Error

}
