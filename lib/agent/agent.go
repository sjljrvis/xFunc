package agent

import (
	"codexec/lib"
	dockerexecutor "codexec/lib/dockerExecutor"
	"codexec/types"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

const (
	red    = "\033[35m"
	italic = "\033[3m"
	reset  = "\033[0m"
)

type AgentAdapter struct {
	types.CoderAgent
}

func New() *AgentAdapter {
	return &AgentAdapter{}
}

func checkTermination(msg string) bool {
	return strings.Contains(msg, "TERMINATE")
}

func (coder *AgentAdapter) StartTimer() {
	timeStamp := time.Now().Unix()
	coder.Instrumentation.TimeTaken = timeStamp
}

func (coder *AgentAdapter) EndTimer() {
	timeStamp := time.Now().Unix()
	coder.Instrumentation.TimeTaken = timeStamp - coder.Instrumentation.TimeTaken
}

func (coder *AgentAdapter) TrackTokens(msg string) {
	t := llms.CountTokens(coder.LLMModel, msg)
	coder.Instrumentation.LLMTokens += t
}

func (coder *AgentAdapter) Run() {
	var roundTrip int32
	ctx := coder.Context
	log.Printf("[CODER] (%d) running task", coder.Task.Id)
	llm, err := openai.New(openai.WithModel(coder.LLMModel))
	if err != nil {
		log.Fatal(err)
	}

	coder.Conversation = append(coder.Conversation, llms.TextParts(llms.ChatMessageTypeSystem, coder.SystemPrompt))
	coder.Conversation = append(coder.Conversation, llms.TextParts(llms.ChatMessageTypeHuman, coder.UserPrompt))
conversationStart:
	select {
	case <-ctx.Done():
		log.Printf("[CODER] Agent stopping due to cancel request task(%d)", coder.Task.Id)
		break
	default:
		coder.Logger.Println("-------------------------------------------------------------------------------------------------------")
		coder.Logger.Println("[CODER] : Thinking ...")
		var buffer strings.Builder
		completion, _ := llm.GenerateContent(ctx, coder.Conversation, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {

			buffer.Write(chunk)
			if strings.Contains(string(chunk), "\n") || strings.Contains(string(chunk), "TERMINATE") {
				// _msg := strings.TrimRight(buffer.String(), "\n")
				// coder.Logger.Print(red, italic, _msg, reset)
				buffer.Reset()
			}
			return nil
		}))

		if ctx.Err() == nil {
			fmt.Println(ctx.Err())
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
					Context:          coder.Context,
					Cancel:           coder.Canel,
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
				coder.Task.CompleteSignal <- true
			}
		}
	}
}
