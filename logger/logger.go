package logger

import (
	pb "codexec/protos/go"
	"log"
)

var Logger *log.Logger

type StreamWriter struct {
	stream pb.StreamService_StreamDataServer // The gRPC stream
}

type CodeStreamWriter struct {
	stream pb.CoderService_ExecuteCodeServer // The gRPC stream
}

func NewStreamWriter(stream pb.StreamService_StreamDataServer) *StreamWriter {
	return &StreamWriter{stream: stream}
}

func NewCodeStreamWriter(stream pb.CoderService_ExecuteCodeServer) *CodeStreamWriter {
	return &CodeStreamWriter{stream: stream}
}

func (w *StreamWriter) Write(p []byte) (n int, err error) {
	message := string(p)
	err = w.stream.Send(&pb.StreamResponse{
		Data: message,
	})

	if err != nil {
		return 0, err
	}

	return len(p), nil
}

func (w *CodeStreamWriter) Write(p []byte) (n int, err error) {
	message := string(p)
	err = w.stream.Send(&pb.CodeResponse{
		Data: message,
	})

	if err != nil {
		return 0, err
	}

	return len(p), nil
}
