package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	errorsx "todo_list/internal/errors"
	"todo_list/internal/models"
	"todo_list/internal/service"
	"todo_list/internal/service/mocks"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newSvc(repo *mocks.MockTodoRepository, cache *mocks.MockCache, producer *mocks.MockKafkaProducer) service.TodoListService {
	return service.NewTodoListService(repo, cache, producer)
}

func ctx() context.Context {
	return context.Background()
}

func TestCreateTodoList_Success(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	repo.On("CreateTodolist", mock.AnythingOfType("models.Todo"), mock.Anything).Return(nil)
	cache.On("Del", mock.Anything, "todos:all").Return(nil)
	producer.On("SendTodoEvent", mock.Anything, "todo_created", mock.Anything).Return(nil)

	svc := newSvc(repo, cache, producer)
	todo, err := svc.CreateTodoList("Buy milk", "Go to store", ctx())

	require.NoError(t, err)
	assert.Equal(t, "Buy milk", todo.Title)
	assert.Equal(t, "Go to store", todo.Description)
	assert.False(t, todo.Completed)
	assert.NotEmpty(t, todo.ID)
	repo.AssertExpectations(t)
	producer.AssertExpectations(t)
}

func TestCreateTodoList_EmptyTitle(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	svc := newSvc(repo, cache, producer)
	_, err := svc.CreateTodoList("", "description", ctx())

	require.Error(t, err)
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INVALID_INPUT", appErr.Code)
	repo.AssertNotCalled(t, "CreateTodolist")
}

func TestCreateTodoList_DBError(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	repo.On("CreateTodolist", mock.Anything, mock.Anything).Return(errors.New("db error"))

	svc := newSvc(repo, cache, producer)
	_, err := svc.CreateTodoList("Buy milk", "desc", ctx())

	require.Error(t, err)
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INTERNAL", appErr.Code)
	producer.AssertNotCalled(t, "SendTodoEvent")
}

func TestCreateTodoList_KafkaError_DoesNotFail(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	repo.On("CreateTodolist", mock.Anything, mock.Anything).Return(nil)
	cache.On("Del", mock.Anything, "todos:all").Return(nil)
	producer.On("SendTodoEvent", mock.Anything, "todo_created", mock.Anything).Return(errors.New("kafka unavailable"))

	svc := newSvc(repo, cache, producer)
	todo, err := svc.CreateTodoList("Buy milk", "desc", ctx())

	require.NoError(t, err)
	assert.Equal(t, "Buy milk", todo.Title)
}

func TestGetTodoList_Success(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	expected := models.Todo{ID: "abc", Title: "Buy milk", Completed: false}
	repo.On("GetTodolist", "abc", mock.Anything).Return(expected, nil)

	svc := newSvc(repo, cache, producer)
	todo, err := svc.GetTodoList("abc", ctx())

	require.NoError(t, err)
	assert.Equal(t, "abc", todo.ID)
	assert.Equal(t, "Buy milk", todo.Title)
}

func TestGetTodoList_NotFound(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	repo.On("GetTodolist", "missing", mock.Anything).Return(models.Todo{}, gorm.ErrRecordNotFound)

	svc := newSvc(repo, cache, producer)
	_, err := svc.GetTodoList("missing", ctx())

	require.Error(t, err)
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_NOT_FOUND", appErr.Code)
}

func TestGetTodoList_DBError(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	repo.On("GetTodolist", "abc", mock.Anything).Return(models.Todo{}, errors.New("connection lost"))

	svc := newSvc(repo, cache, producer)
	_, err := svc.GetTodoList("abc", ctx())

	require.Error(t, err)
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INTERNAL", appErr.Code)
}

func TestGetAllTodoList_FromCache(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	cached := `[{"id":"abc","title":"Buy milk","description":"","completed":false,"created_at":"0001-01-01T00:00:00Z","updated_at":null}]`
	cache.On("Get", mock.Anything, "todos:all").Return(cached, nil)

	svc := newSvc(repo, cache, producer)
	todos, err := svc.GetAllTodoList(ctx())

	require.NoError(t, err)
	assert.Len(t, todos, 1)
	assert.Equal(t, "Buy milk", todos[0].Title)
	repo.AssertNotCalled(t, "GetAllTodolist")
}

func TestGetAllTodoList_CacheMiss_FallbackToDB(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	cache.On("Get", mock.Anything, "todos:all").Return("", redis.Nil)
	repo.On("GetAllTodolist", mock.Anything).Return([]models.Todo{
		{ID: "abc", Title: "Buy milk"},
	}, nil)
	cache.On("Set", mock.Anything, "todos:all", mock.Anything, 120*time.Second).Return(nil)

	svc := newSvc(repo, cache, producer)
	todos, err := svc.GetAllTodoList(ctx())

	require.NoError(t, err)
	assert.Len(t, todos, 1)
}

func TestGetAllTodoList_DBError(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	cache.On("Get", mock.Anything, "todos:all").Return("", redis.Nil)
	repo.On("GetAllTodolist", mock.Anything).Return([]models.Todo{}, errors.New("db error"))

	svc := newSvc(repo, cache, producer)
	_, err := svc.GetAllTodoList(ctx())

	require.Error(t, err)
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INTERNAL", appErr.Code)
}

func TestUpdateTodoList_Success(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	existing := models.Todo{ID: "abc", Title: "Old title", Description: "Old desc"}
	repo.On("GetTodolist", "abc", mock.Anything).Return(existing, nil)
	repo.On("UpdateTodolist", mock.AnythingOfType("models.Todo"), mock.Anything).Return(nil)
	producer.On("SendTodoEvent", mock.Anything, "todo_updated", mock.Anything).Return(nil)
	cache.On("Del", mock.Anything, "todos:all").Return(nil)

	svc := newSvc(repo, cache, producer)
	todo, err := svc.UpdateTodoList("abc", "New title", "New desc", ctx())

	require.NoError(t, err)
	assert.Equal(t, "New title", todo.Title)
	assert.Equal(t, "New desc", todo.Description)
	assert.NotNil(t, todo.UpdatedAt)
}

func TestUpdateTodoList_EmptyTitle(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	svc := newSvc(repo, cache, producer)
	_, err := svc.UpdateTodoList("abc", "", "desc", ctx())

	require.Error(t, err)
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INVALID_INPUT", appErr.Code)
	repo.AssertNotCalled(t, "GetTodolist")
}

func TestUpdateTodoList_NotFound(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	repo.On("GetTodolist", "missing", mock.Anything).Return(models.Todo{}, gorm.ErrRecordNotFound)

	svc := newSvc(repo, cache, producer)
	_, err := svc.UpdateTodoList("missing", "New title", "desc", ctx())

	require.Error(t, err)
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_NOT_FOUND", appErr.Code)
}

func TestDeleteTodoList_Success(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	repo.On("DeleteTodolist", "abc", mock.Anything).Return(nil)
	producer.On("SendTodoEvent", mock.Anything, "todo_deleted", mock.Anything).Return(nil)
	cache.On("Del", mock.Anything, "todos:all").Return(nil)

	svc := newSvc(repo, cache, producer)
	err := svc.DeleteTodoList("abc", ctx())

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeleteTodoList_NotFound(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	repo.On("DeleteTodolist", "missing", mock.Anything).Return(gorm.ErrRecordNotFound)

	svc := newSvc(repo, cache, producer)
	err := svc.DeleteTodoList("missing", ctx())

	require.Error(t, err)
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_NOT_FOUND", appErr.Code)
	producer.AssertNotCalled(t, "SendTodoEvent")
}

func TestMarkTodoCompleted_Success(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	existing := models.Todo{ID: "abc", Title: "Buy milk", Completed: false}
	repo.On("GetTodolist", "abc", mock.Anything).Return(existing, nil)
	repo.On("UpdateTodolist", mock.AnythingOfType("models.Todo"), mock.Anything).Return(nil)
	cache.On("Del", mock.Anything, "todos:all").Return(nil)

	svc := newSvc(repo, cache, producer)
	todo, err := svc.MarkTodoCompleted("abc", true, ctx())

	require.NoError(t, err)
	assert.True(t, todo.Completed)
	assert.NotNil(t, todo.UpdatedAt)
}

func TestMarkTodoCompleted_NotFound(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	repo.On("GetTodolist", "missing", mock.Anything).Return(models.Todo{}, gorm.ErrRecordNotFound)

	svc := newSvc(repo, cache, producer)
	_, err := svc.MarkTodoCompleted("missing", true, ctx())

	require.Error(t, err)
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_NOT_FOUND", appErr.Code)
}

func TestMarkTodoCompleted_Idempotent(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	existing := models.Todo{ID: "abc", Title: "Buy milk", Completed: true}
	repo.On("GetTodolist", "abc", mock.Anything).Return(existing, nil)
	repo.On("UpdateTodolist", mock.Anything, mock.Anything).Return(nil)
	cache.On("Del", mock.Anything, "todos:all").Return(nil)

	svc := newSvc(repo, cache, producer)
	todo, err := svc.MarkTodoCompleted("abc", true, ctx())

	require.NoError(t, err)
	assert.True(t, todo.Completed)
}

func TestDeleteAllTodolist_Success(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	repo.On("DeleteAllTodolist", mock.Anything).Return(nil)
	cache.On("Del", mock.Anything, "todos:all").Return(nil)

	svc := newSvc(repo, cache, producer)
	err := svc.DeleteAllTodolist(ctx())

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeleteAllTodolist_DBError(t *testing.T) {
	repo := &mocks.MockTodoRepository{}
	cache := &mocks.MockCache{}
	producer := &mocks.MockKafkaProducer{}

	repo.On("DeleteAllTodolist", mock.Anything).Return(errors.New("db error"))

	svc := newSvc(repo, cache, producer)
	err := svc.DeleteAllTodolist(ctx())

	require.Error(t, err)
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INTERNAL", appErr.Code)
}

func TestValidateTodoInput_EmptyTitle(t *testing.T) {
	err := service.ValidateTodoInput("")
	require.Error(t, err)
	var appErr *errorsx.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ERR_INVALID_INPUT", appErr.Code)
}

func TestValidateTodoInput_ValidTitle(t *testing.T) {
	err := service.ValidateTodoInput("Buy milk")
	require.NoError(t, err)
}
