package rpc

import (
	"codexec/logger"
	pb "codexec/protos/go"
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
)

type CoderServiceServer struct {
	pb.UnimplementedCoderServiceServer
	workerPool *WorkerPool
}

func (s *CoderServiceServer) ExecuteCode(req *pb.CodeRequest, stream pb.CoderService_ExecuteCodeServer) error {
	completeSignal := make(chan bool, 1)
	defer close(completeSignal)

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	streamWriter := logger.NewCodeStreamWriter(stream)
	streamLogger := log.New(streamWriter, "", 0)

	task := Task{
		id:               GenerateRandomID(),
		completeSignal:   completeSignal,
		systemPrompt:     req.SystemPrompt,
		userPrompt:       req.UserPrompt,
		workingDirectory: req.WorkingDirectory,
		dockerImage:      req.DockerImage,
		maxRetry:         req.MaxRetry,
		llmModel:         req.LLMModel,
		Logger:           streamLogger,
		Context:          ctx,
		Cancel:           cancel,
	}

	s.workerPool.SubmitTask(task)

	select {
	case <-completeSignal:
		return nil
	case <-ctx.Done():
		// Client disconnected
		log.Printf("Client disconnected, cancelling task %d", task.id)
		return ctx.Err()
	}

	// <-completeSignal
	// return nil
}

func StartRPCServer() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	workerPool := NewWorkerPool(2)

	defer workerPool.Close()

	grpcServer := grpc.NewServer()

	pb.RegisterCoderServiceServer(grpcServer, &CoderServiceServer{
		workerPool: workerPool,
	})

	log.Println("gRPC server running on port 50051...")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
