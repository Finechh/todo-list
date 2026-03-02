package app

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"todo_list/internal/database"
	grpcTodo "todo_list/internal/grpc"
	todopb "todo_list/internal/grpc/proto/pb"
	"todo_list/internal/handlers"
	"todo_list/internal/kafka/audit"
	handlerAudit "todo_list/internal/kafka/audit/handlers"
	auditRepo "todo_list/internal/kafka/audit/repository"
	"todo_list/internal/kafka/consumer"
	"todo_list/internal/kafka/producer"
	middlewarex "todo_list/internal/middleware"
	cache "todo_list/internal/redis"
	"todo_list/internal/repository"
	"todo_list/internal/service"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

func Run() error {
	ctx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer initCancel()

	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("could not connect to DB: %v", err)
	}

	redisCache, err := cache.NewCache(ctx, "localhost:6379")
	if err != nil {
		log.Fatalf("cannot connect to Redis: %v", err)
	}

	kafkaProducer, err := producer.NewProducer([]string{"localhost:9092"}, "todos")
	if err != nil {
		log.Fatalf("failed to init kafka producer: %v", err)
	}

	ctxKafka, cancelKafka := context.WithCancel(context.Background())

	auditRepository := auditRepo.NewAuditRepo(db)
	auditHandler := handlerAudit.NewHandler(auditRepository)
	auditProducer := audit.NewKafkaTodoEvent(kafkaProducer, "todos")

	cons, err := consumer.New([]string{"localhost:9092"}, "audit-group", auditHandler, ctxKafka)
	if err != nil {
		log.Fatalf("failed to create kafka consumer: %v", err)
	}

	consumerDone := make(chan struct{})
	go func() {
		defer close(consumerDone)
		log.Println("kafka consumer started")
		if err := cons.Run("todos"); err != nil && ctxKafka.Err() == nil {
			log.Printf("kafka consumer error: %v", err)
		}
		log.Println("kafka consumer stopped")
	}()

	todoRepo := repository.NewTodoListRepository(db)
	todoService := service.NewTodoListService(todoRepo, redisCache, auditProducer)
	todoHandlers := handlers.NewTodoListHandlers(todoService)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen on :50051: %v", err)
	}
	grpcServer := grpc.NewServer()
	todopb.RegisterTodoServiceServer(grpcServer, &grpcTodo.TodoGRPCServer{
		Service: todoService,
	})

	grpcDone := make(chan struct{})
	go func() {
		defer close(grpcDone)
		log.Println("gRPC server started on :50051")
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server stopped: %v", err)
		}
	}()

	e := echo.New()
	e.HideBanner = true
	e.Use(middlewarex.RequestID)
	e.Use(middlewarex.LoggerMiddleware)
	e.Use(middlewarex.ErrorHandler)

	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	e.GET("/posts", todoHandlers.GetTodoList)
	e.POST("/posts", todoHandlers.PostTodoList)
	e.PATCH("/posts/:id", todoHandlers.PatchTodoList)
	e.PATCH("/posts/:id/completed", todoHandlers.MarkTodoCompleted)
	e.DELETE("/posts/:id", todoHandlers.DeleteTodoList)
	e.DELETE("/posts", todoHandlers.DeleteAllTodolist)

	httpErr := make(chan error, 1)
	go func() {
		log.Println("HTTP server started on localhost:8080")
		if err := e.Start("localhost:8080"); err != nil {
			httpErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Printf("received signal: %s, shutting down...", sig)
	case err := <-httpErr:
		log.Printf("HTTP server error: %v", err)
	}

	log.Println("stopping kafka consumer...")
	cancelKafka()
	select {
	case <-consumerDone:
		log.Println("kafka consumer stopped")
	case <-time.After(5 * time.Second):
		log.Println("kafka consumer shutdown timeout")
	}

	log.Println("stopping gRPC server...")
	grpcServer.GracefulStop()
	<-grpcDone
	log.Println("gRPC server stopped")

	log.Println("stopping HTTP server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	log.Println("HTTP server stopped")

	if err := redisCache.Close(); err != nil {
		log.Printf("redis close error: %v", err)
	}
	log.Println("shutdown complete")
	return nil
}