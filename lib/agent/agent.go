package agent

import (
	"codexec/lib"
	dockerexecutor "codexec/lib/dockerExecutor"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
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
}

func checkTermination(msg string) bool {
	return strings.Contains(msg, "TERMINATE")
}

const (
	red    = "\033[35m"
	italic = "\033[3m"
	reset  = "\033[0m"
)

func (coder *CoderAgent) StartTimer() {
	timeStamp := time.Now().Unix()
	coder.Instrumentation.TimeTaken = timeStamp
}

func (coder *CoderAgent) EndTimer() {
	timeStamp := time.Now().Unix()
	coder.Instrumentation.TimeTaken = timeStamp - coder.Instrumentation.TimeTaken
}

func (coder *CoderAgent) TrackTokens(msg string) {
	t := llms.CountTokens(coder.LLMModel, msg)
	coder.Instrumentation.LLMTokens += t
}

func (coder *CoderAgent) Run() {
	var roundTrip int32

	llm, err := openai.New(openai.WithModel(coder.LLMModel))
	if err != nil {
		log.Fatal(err)
	}

	ctx := coder.Context // context.Background()
	// Initialize conversation with system message

	go func() {
		select {
		case <-ctx.Done():
			fmt.Println("access DB task1 error:", ctx.Err())
			return
		}
	}()

	coder.Conversation = append(coder.Conversation, llms.TextParts(llms.ChatMessageTypeSystem, coder.SystemPrompt))
	coder.Conversation = append(coder.Conversation, llms.TextParts(llms.ChatMessageTypeHuman, coder.UserPrompt))

conversationStart:
	coder.Logger.Println("-------------------------------------------------------------------------------------------------------")
	coder.Logger.Println("[CODER] : Thinking ...")
	var buffer strings.Builder
	completion, err := llm.GenerateContent(ctx, coder.Conversation, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {

		buffer.Write(chunk)
		if strings.Contains(string(chunk), "\n") || strings.Contains(string(chunk), "TERMINATE") {
			// _msg := strings.TrimRight(buffer.String(), "\n")
			// coder.Logger.Print(red, italic, _msg, reset)
			buffer.Reset() // Clear the buffer after logging
		}
		return nil
	}))

	fmt.Println(err)
	msgContent := completion.Choices[0].Content
	coder.TrackTokens(msgContent)

	if !checkTermination(msgContent) {
		coder.Logger.Print(red, italic, msgContent, reset)
		// Add AI response to conversation
		coder.Conversation = append(coder.Conversation, llms.TextParts(llms.ChatMessageTypeAI, msgContent))
		// Extract Code blocks
		coder.Logger.Printf("[EXECUTOR] [retry: %d]: %s\n\n", roundTrip, "Extracting Code blocks")
		lib.SplitIntoCodeBlocksAndSave(msgContent, coder.WorkingDirectory)
		coder.Logger.Printf("[EXECUTOR] [retry: %d]: %s\n\n", roundTrip, "Executing Code blocks")

		dockerExecuteParams := dockerexecutor.DockerExecuteParams{
			ContainerName:    coder.DockerContainerName,
			WorkingDirectory: coder.WorkingDirectory,
			DockerImage:      coder.DockerImage,
		}

		dockerExecReponse := dockerexecutor.Run(dockerExecuteParams)

		// if dockerExecReponse.ExitCode != 0 {
		// }

		roundTrip = roundTrip + 1
		if roundTrip < coder.MaxRetry {
			if dockerExecReponse.ExitCode != 0 {
				coder.Logger.Printf("[EXECUTOR] [retry: %d]: exit_code -  %d \n\n", roundTrip, dockerExecReponse.ExitCode)
				coder.Logger.Printf("[EXECUTOR] [retry: %d]: %s\n\n", roundTrip, "Give me another example for Code")
				modificationPrompt := fmt.Sprintf("Give me another example with modification, stdout received : %s", dockerExecReponse.Stdout)
				coder.Conversation = append(coder.Conversation, llms.TextParts(llms.ChatMessageTypeHuman, modificationPrompt))
				goto conversationStart
			} else {
				coder.Logger.Printf("[EXECUTOR] [retry: %d]: exit_code - %d, stdout received -  %s  \n\n", roundTrip, dockerExecReponse.ExitCode, dockerExecReponse.Stdout)
				modificationPrompt := fmt.Sprintf(" exit_code - %d, stdout received : %s", dockerExecReponse.ExitCode, dockerExecReponse.Stdout)
				coder.Conversation = append(coder.Conversation, llms.TextParts(llms.ChatMessageTypeHuman, modificationPrompt))
				goto conversationStart
			}
		} else {
			coder.Logger.Println("terminate due to retries")
		}
	} else {
		coder.Logger.Println(red, italic, msgContent, reset)
	}
}
