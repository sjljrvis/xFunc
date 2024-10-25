package rpc

import (
	"codexec/logger"
	pb "codexec/protos/go"
	"codexec/types"
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
)

type CoderServiceServer struct {
	pb.UnimplementedCoderServiceServer
	workerPool *WorkerPoolAdapter
}

func (s *CoderServiceServer) ExecuteCode(req *pb.CodeRequest, stream pb.CoderService_ExecuteCodeServer) error {
	CompleteSignal := make(chan bool, 1)
	defer close(CompleteSignal)

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	streamWriter := logger.NewCodeStreamWriter(stream)
	streamLogger := log.New(streamWriter, "", 0)

	task := types.Task{
		Id:               GenerateRandomID(),
		CompleteSignal:   CompleteSignal,
		SystemPrompt:     req.SystemPrompt,
		UserPrompt:       req.UserPrompt,
		WorkingDirectory: req.WorkingDirectory,
		DockerImage:      req.DockerImage,
		MaxRetry:         req.MaxRetry,
		LLMModel:         req.LLMModel,
		Logger:           streamLogger,
		Context:          ctx,
		Cancel:           cancel,
	}

	s.workerPool.SubmitTask(task)
	select {
	case <-CompleteSignal:
		log.Printf("[WORKER] (%d) finished", task.Id)
		return nil
	case <-ctx.Done():
		log.Printf("[RPC] client disconnected, cancelling task (%d)", task.Id)
		return ctx.Err()
	}
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
