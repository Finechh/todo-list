package grpcTodo

import (
	"context"
	"errors"
	"todo_list/internal/errors"
	"todo_list/internal/grpc/proto/pb"
	"todo_list/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TodoGRPCServer struct {
	todopb.UnimplementedTodoServiceServer
	Service service.TodoListService
}

func NewTodoGRPCServer(s service.TodoListService) *TodoGRPCServer {
	return &TodoGRPCServer{Service: s}
}

func (t *TodoGRPCServer) CreateTodo(ctx context.Context, req *todopb.CreateTodoRequest) (*todopb.CreateTodoResponse, error) {
	todo, err := t.Service.CreateTodoList(req.Title, req.Description, ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &todopb.CreateTodoResponse{
		Todo: &todopb.Todo{
			Id:          todo.ID,
			Title:       todo.Title,
			Description: todo.Description,
			Completed:   todo.Completed,
		},
	}, nil
}

func (t *TodoGRPCServer) GetTodo(ctx context.Context, req *todopb.GetTodoRequest) (*todopb.GetTodoResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	todo, err := t.Service.GetTodoList(req.Id, ctx)
	if err != nil {
		return nil, status.Error(codes.NotFound, "todo not found")
	}

	return &todopb.GetTodoResponse{
		Todo: &todopb.Todo{
			Id:          todo.ID,
			Title:       todo.Title,
			Description: todo.Description,
			Completed:   todo.Completed,
		},
	}, nil
}

func (t *TodoGRPCServer) MarkTodoCompleted(ctx context.Context, req *todopb.MarkTodoCompletedRequest) (*todopb.MarkTodoCompletedResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	todo, err := t.Service.MarkTodoCompleted(req.Id, req.Completed, ctx)
	if err != nil {
		var appErr *errorsx.AppError
		if errors.As(err, &appErr) {
			switch appErr.Code {
			case "ERR_NOT_FOUND":
				return nil, status.Error(codes.NotFound, appErr.Message)
			case "ERR_INVALID_INPUT":
				return nil, status.Error(codes.InvalidArgument, appErr.Message)
			default:
				return nil, status.Error(codes.Internal, appErr.Message)
			}
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &todopb.MarkTodoCompletedResponse{
		Todo: &todopb.Todo{
			Id:          todo.ID,
			Title:       todo.Title,
			Description: todo.Description,
			Completed:   todo.Completed,
		},
	}, nil
}

func (t *TodoGRPCServer) UpdateTodo(ctx context.Context, req *todopb.UpdateTodoRequest) (*todopb.UpdateTodoResponse, error) {
	if req.Id == "" || req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "id and title are required")
	}

	todo, err := t.Service.UpdateTodoList(req.Id, req.Title, req.Description, ctx)
	if err != nil {
		var appErr *errorsx.AppError
		if errors.As(err, &appErr) {
			switch appErr.Code {
			case "ERR_NOT_FOUND":
				return nil, status.Error(codes.NotFound, appErr.Message)
			case "ERR_INVALID_INPUT":
				return nil, status.Error(codes.InvalidArgument, appErr.Message)
			default:
				return nil, status.Error(codes.Internal, appErr.Message)
			}
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &todopb.UpdateTodoResponse{
		Todo: &todopb.Todo{
			Id:          todo.ID,
			Title:       todo.Title,
			Description: todo.Description,
			Completed:   todo.Completed,
		},
	}, nil
}
