package handlers

import (
	"net/http"
	"time"
	errorsx "todo_list/internal/errors"
	"todo_list/internal/metrics"
	middlewarex "todo_list/internal/middleware"
	"todo_list/internal/models"
	"todo_list/internal/service"

	"github.com/labstack/echo/v4"
)

type TodoListHandlers struct {
	service service.TodoListService
}

func NewTodoListHandlers(s service.TodoListService) *TodoListHandlers {
	return &TodoListHandlers{service: s}
}

func (h *TodoListHandlers) GetTodoList(c echo.Context) error {
	start := time.Now()
	defer metrics.ObserveHttpDuration(c.Request().Method, c.Path(), time.Since(start))

	ctx, cancel := middlewarex.RequestCtx(c, 2)
	defer cancel()

	tdl, err := h.service.GetAllTodoList(ctx)
	if err != nil {
		metrics.IncHttpErrorTotal(c.Request().Method, c.Path())
		return err
	}

	metrics.IncHttpRequestTotal(c.Request().Method, c.Path())

	return c.JSON(http.StatusOK, tdl)
}

func (h *TodoListHandlers) PostTodoList(c echo.Context) error {
	start := time.Now()
	defer metrics.ObserveHttpDuration(c.Request().Method, c.Path(), time.Since(start))

	var req models.CreateTodoRequest
	if err := c.Bind(&req); err != nil {
		metrics.IncHttpErrorTotal(c.Request().Method, c.Path())
		return errorsx.ErrInvalidInput("invalid request body")
	}
	if req.Description == "" {
		req.Description = "No description"
	}
	ctx, cancel := middlewarex.RequestCtx(c, 2)
	defer cancel()
	tdl, err := h.service.CreateTodoList(req.Title, req.Description, ctx)
	if err != nil {
		metrics.IncHttpErrorTotal(c.Request().Method, c.Path())
		return err
	}

	metrics.IncHttpRequestTotal(c.Request().Method, c.Path())

	return c.JSON(http.StatusOK, tdl)
}

func (h *TodoListHandlers) PatchTodoList(c echo.Context) error {
	start := time.Now()
	defer metrics.ObserveHttpDuration(c.Request().Method, c.Path(), time.Since(start))

	id := c.Param("id")
	var req models.UpdateTodoRequest
	if err := c.Bind(&req); err != nil {
		metrics.IncHttpErrorTotal(c.Request().Method, c.Path())
		return errorsx.ErrInvalidInput("invalid request body")
	}
	ctx, cancel := middlewarex.RequestCtx(c, 2)
	defer cancel()
	update, err := h.service.UpdateTodoList(id, req.Title, req.Description, ctx)
	if err != nil {
		metrics.IncHttpErrorTotal(c.Request().Method, c.Path())
		return err
	}

	metrics.IncHttpRequestTotal(c.Request().Method, c.Path())
	return c.JSON(http.StatusOK, update)
}

func (h *TodoListHandlers) MarkTodoCompleted(c echo.Context) error {
	start := time.Now()
	defer metrics.ObserveHttpDuration(c.Request().Method, c.Path(), time.Since(start))

	id := c.Param("id")
	var req models.UpdateTodoRequest
	if err := c.Bind(&req); err != nil {
		metrics.IncHttpErrorTotal(c.Request().Method, c.Path())
		return errorsx.ErrInvalidInput("invalid request body")
	}

	ctx, cancel := middlewarex.RequestCtx(c, 2)
	defer cancel()

	todo, err := h.service.MarkTodoCompleted(id, req.Completed, ctx)
	if err != nil {
		metrics.IncHttpErrorTotal(c.Request().Method, c.Path())
		return err
	}

	metrics.IncHttpRequestTotal(c.Request().Method, c.Path())

	return c.JSON(http.StatusOK, todo)
}

func (h *TodoListHandlers) DeleteTodoList(c echo.Context) error {
	start := time.Now()
	defer metrics.ObserveHttpDuration(c.Request().Method, c.Path(), time.Since(start))

	ctx, cancel := middlewarex.RequestCtx(c, 2)
	defer cancel()

	id := c.Param("id")
	if err := h.service.DeleteTodoList(id, ctx); err != nil {
		metrics.IncHttpErrorTotal(c.Request().Method, c.Path())
		return err
	}

	metrics.IncHttpRequestTotal(c.Request().Method, c.Path())

	return c.NoContent(http.StatusNoContent)
}

func (h *TodoListHandlers) DeleteAllTodolist(c echo.Context) error {
	start := time.Now()
	defer metrics.ObserveHttpDuration(c.Request().Method, c.Path(), time.Since(start))

	ctx, cancel := middlewarex.RequestCtx(c, 3)
	defer cancel()
	err := h.service.DeleteAllTodolist(ctx)
	if err != nil {
		metrics.IncHttpErrorTotal(c.Request().Method, c.Path())
		return err
	}

	metrics.IncHttpRequestTotal(c.Request().Method, c.Path())

	return c.NoContent(http.StatusNoContent)
}
