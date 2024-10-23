package rpc

import (
	"codexec/config"
	"codexec/lib"
	"codexec/lib/agent"
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

type Task struct {
	id               int
	systemPrompt     string
	userPrompt       string
	workingDirectory string
	dockerImage      string
	maxRetry         int32
	llmModel         string
	completeSignal   chan<- bool
	Logger           *log.Logger
	Context          context.Context
	Cancel           context.CancelFunc
	// response         chan<- *pb.CodeResponse
}

type WorkerPool struct {
	tasks chan Task
	wg    sync.WaitGroup
}

func NewWorkerPool(numWorkers int) *WorkerPool {
	pool := &WorkerPool{
		tasks: make(chan Task),
	}
	for i := 1; i <= numWorkers; i++ {
		pool.wg.Add(1)
		go pool.worker(i)
	}
	return pool
}

func (p *WorkerPool) SubmitTask(task Task) {
	p.tasks <- task
}

func (p *WorkerPool) Close() {
	close(p.tasks)
	p.wg.Wait()
}

func GenerateRandomID() int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(1000000) + 1
}

func (p *WorkerPool) worker(workerID int) {
	defer p.wg.Done()
	for task := range p.tasks {

		select {
		case <-task.Context.Done():
			log.Printf("Task %d cancelled before execution", task.id)
			task.completeSignal <- true
			continue
		default:
		}

		containerName := lib.GetContainerName(12)
		// Creating a blank dir
		// hostDir := fmt.Sprintf("%s%s", config.Data.Get("app.codingDirectory").(string), task.workingDirectory)
		hostDir := fmt.Sprintf("%s%s", config.Data.Get("app.codingDirectory").(string), containerName)

		coder := agent.CoderAgent{
			SystemPrompt:        task.systemPrompt,
			UserPrompt:          task.userPrompt,
			DockerImage:         task.dockerImage,
			DockerContainerName: containerName,
			LLMModel:            task.llmModel,
			MaxRetry:            task.maxRetry,
			WorkingDirectory:    hostDir,
			MaxTimeOut:          1,
			Logger:              task.Logger,
			Instrumentation:     agent.InstrumentationStats{},
			Context:             task.Context,
		}

		coder.StartTimer()

		go func() {
			select {
			case <-task.Context.Done():
				log.Printf("Task %d cancelled during execution", task.id)
				// Perform any necessary cleanup here
				coder.EndTimer()
				task.completeSignal <- true
			}
		}()
		coder.Run()
		coder.EndTimer()
		task.completeSignal <- true
	}
}
