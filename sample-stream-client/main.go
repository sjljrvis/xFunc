package main

import (
	"context"
	"io"
	"log"

	pb "codexec/protos/go"

	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewStreamServiceClient(conn)

	req := &pb.StreamRequest{
		Query: "example",
	}

	// Call the streaming RPC
	stream, err := client.StreamData(context.Background(), req)
	if err != nil {
		log.Fatalf("Failed to call StreamData: %v", err)
	}

	for {
		response, err := stream.Recv()
		if err == io.EOF {
			// Handle end of stream
			log.Println("Stream closed by server.")
			break
		}
		if err != nil {
			log.Fatalf("Error while receiving stream: %v", err)
		}
		log.Println("Received:", response.Data)
	}
}
