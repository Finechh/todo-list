package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"todo_list/internal/errors"
	"todo_list/internal/kafka/audit"
	middlewarex "todo_list/internal/middleware"
	s "todo_list/internal/models"
	cache "todo_list/internal/redis"
	"todo_list/internal/repository"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type TodoListService interface {
	CreateTodoList(Title, Description string, ctx context.Context) (s.Todo, error)
	GetTodoList(id string, ctx context.Context) (s.Todo, error)
	GetAllTodoList(ctx context.Context) ([]s.Todo, error)
	UpdateTodoList(id, title, description string, ctx context.Context) (s.Todo, error)
	DeleteTodoList(id string, ctx context.Context) error
	MarkTodoCompleted(id string, completed bool, ctx context.Context) (s.Todo, error)
	DeleteAllTodolist(ctx context.Context) error
}

type TodoService struct {
	repo     repository.TodolistRepository
	rdb      cache.CacheStore
	producer audit.TodoEventProducer
}

func NewTodoListService(r repository.TodolistRepository, c cache.CacheStore, p audit.TodoEventProducer) TodoListService {
	return &TodoService{repo: r, rdb: c, producer: p}
}

func ReqIDFromCtx(ctx context.Context) string {
	id, ok := ctx.Value(middlewarex.RqID).(string)
	if !ok {
		return "unknown"
	}
	return id
}

func (r *TodoService) MarkTodoCompleted(id string, completed bool, ctx context.Context) (s.Todo, error) {
	todo, err := r.repo.GetTodolist(id, ctx)
	if err != nil {
		return s.Todo{}, errorsx.ErrNotFound("todo not found")
	}
	now := time.Now()
	todo.Completed = completed
	todo.UpdatedAt = &now

	if err := r.repo.UpdateTodolist(todo, ctx); err != nil {
		log.Printf("db update error: %v, reqID = %s", err, ReqIDFromCtx(ctx))
		return s.Todo{}, errorsx.ErrInternalError("failed to update todo")
	}
	if err := r.rdb.Del(ctx, "todos:all"); err != nil {
		log.Printf("cache del error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
	}
	return todo, nil
}

func ValidateTodoInput(title string) error {
	if len(title) == 0 {
		return errorsx.ErrInvalidInput("title cannot be empty")
	}
	return nil
}

func (r *TodoService) CreateTodoList(Title, Description string, ctx context.Context) (s.Todo, error) {
	if err := ValidateTodoInput(Title); err != nil {
		return s.Todo{}, err
	}

	model := s.Todo{
		ID:          uuid.NewString(),
		Title:       Title,
		Description: Description,
		Completed:   false,
		UpdatedAt:   nil,
	}

	if err := r.repo.CreateTodolist(model, ctx); err != nil {
		log.Printf("db create error: %v, reqID = %s", err, ReqIDFromCtx(ctx))
		return s.Todo{}, errorsx.ErrInternalError("failed to create todo")
	}

	jsonData, err := json.Marshal(model)
	if err != nil {
		log.Printf("%s", err)
	}

	if err := r.producer.SendTodoEvent(ctx, "todo_created", jsonData); err != nil {
		log.Printf("[kafka] failed to send todo_created event: %v", err)
	}

	if err := r.rdb.Del(ctx, "todos:all"); err != nil {
		log.Printf("cache del error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
	}
	return model, nil
}

func (r *TodoService) DeleteTodoList(id string, ctx context.Context) error {
	if err := r.repo.DeleteTodolist(id, ctx); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("db (not found) error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
			return errorsx.ErrNotFound("todo not found")
		}
		log.Printf("db del error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
		return errorsx.ErrInternalError("failed to delete todo")
	}

	jsonData, err := json.Marshal(id)
	if err != nil {
		log.Printf("%s", err)
	}
	if err := r.producer.SendTodoEvent(ctx, "todo_deleted", jsonData); err != nil {
		log.Printf("[kafka] failed to send todo_deleted event: %v", err)
	}

	if err := r.rdb.Del(ctx, "todos:all"); err != nil {
		log.Printf("cache del error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
	}
	return nil
}

func (r *TodoService) GetAllTodoList(ctx context.Context) ([]s.Todo, error) {
	cacheKey := "todos:all"
	cached, err := r.rdb.Get(ctx, cacheKey)
	if err == nil {
		var todos []s.Todo
		if err := json.Unmarshal([]byte(cached), &todos); err == nil {
			return todos, nil
		} else {
			log.Printf("cache unmarshal error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
		}
	} else if err != redis.Nil {
		log.Printf("cache get error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
	}

	todos, err := r.repo.GetAllTodolist(ctx)
	if err != nil {
		log.Printf("db getAll error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
		return nil, errorsx.ErrInternalError("failed to load todos from bd")
	}

	jsonData, err := json.Marshal(todos)
	if err == nil {
		if err := r.rdb.Set(ctx, cacheKey, jsonData, 120*time.Second); err != nil {
			log.Printf("cache set error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
		}
	} else {
		log.Printf("json marshal error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
	}
	return todos, nil
}

func (r *TodoService) GetTodoList(id string, ctx context.Context) (s.Todo, error) {
	todo, err := r.repo.GetTodolist(id, ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("db get error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
			return s.Todo{}, errorsx.ErrNotFound("todo not found")
		}
		log.Printf("db get error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
		return s.Todo{}, errorsx.ErrInternalError("database error")
	}
	return todo, nil
}

func (r *TodoService) UpdateTodoList(id, title, description string, ctx context.Context) (s.Todo, error) {
	if err := ValidateTodoInput(title); err != nil {
		log.Printf("db validate error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
		return s.Todo{}, err
	}

	tdl, err := r.repo.GetTodolist(id, ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("db get error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
			return s.Todo{}, errorsx.ErrNotFound("todo not found")
		}
		log.Printf("db get error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
		return s.Todo{}, errorsx.ErrInternalError("database error")
	}

	now := time.Now()
	tdl.Title = title
	tdl.Description = description
	tdl.UpdatedAt = &now

	if err := r.repo.UpdateTodolist(tdl, ctx); err != nil {
		log.Printf("db update error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
		return s.Todo{}, errorsx.ErrInternalError("failed to update todo")
	}

	jsonData, err := json.Marshal(tdl)
	if err != nil {
		log.Printf("%s", err)
	}

	if err := r.producer.SendTodoEvent(ctx, "todo_updated", jsonData); err != nil {
		log.Printf("[kafka] failed to send todo_updated event: %v", err)
	}

	if err := r.rdb.Del(ctx, "todos:all"); err != nil {
		log.Printf("cache del error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
	}
	return tdl, nil
}

func (r *TodoService) DeleteAllTodolist(ctx context.Context) error {
	if err := r.repo.DeleteAllTodolist(ctx); err != nil {
		log.Printf("db delAll error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
		return errorsx.ErrInternalError("failed to delete all todo")
	}
	if err := r.rdb.Del(ctx, "todos:all"); err != nil {
		log.Printf("cache del error: %v, reqID=%s", err, ReqIDFromCtx(ctx))
	}
	return nil
}