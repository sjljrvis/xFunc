package types

import (
	"context"
	"log"
	"sync"

	"github.com/tmc/langchaingo/llms"
)

type InstrumentationStats struct {
	LLMTokens int
	TimeTaken int64
}

type CoderAgent struct {
	SystemPrompt        string
	UserPrompt          string
	DockerImage         string
	DockerContainerName string
	MaxRetry            int32
	LLMModel            string
	WorkingDirectory    string
	MaxTimeOut          int32
	Conversation        []llms.MessageContent
	Logger              *log.Logger
	Instrumentation     InstrumentationStats
	Context             context.Context
	Canel               context.CancelFunc
	Task                *Task
}

type Task struct {
	Id               int
	SystemPrompt     string
	UserPrompt       string
	WorkingDirectory string
	DockerImage      string
	MaxRetry         int32
	LLMModel         string
	CompleteSignal   chan<- bool
	Logger           *log.Logger
	Context          context.Context
	Cancel           context.CancelFunc
}

type WorkerPool struct {
	Tasks chan Task
	Wg    sync.WaitGroup
}
