package rpc

import (
	"codexec/config"
	"codexec/lib"
	"codexec/lib/agent"
	"codexec/types"
	"fmt"
	"log"
	"math/rand"
	"time"
)

type WorkerPoolAdapter struct {
	types.WorkerPool
}

type TaskAdapter struct {
	types.Task
}

func NewWorkerPool(numWorkers int) *WorkerPoolAdapter {
	pool := &WorkerPoolAdapter{
		WorkerPool: types.WorkerPool{
			Tasks: make(chan types.Task),
		},
	}
	for i := 1; i <= numWorkers; i++ {
		pool.Wg.Add(1)
		go pool.worker(i)
	}
	return pool
}

func (t *TaskAdapter) Complete() {
	t.CompleteSignal <- true
}

func (p *WorkerPoolAdapter) SubmitTask(task types.Task) {
	log.Printf("[WORKER] (%d) queued", task.Id)
	p.Tasks <- task
}

func (p *WorkerPoolAdapter) Close() {
	close(p.Tasks)
	p.Wg.Wait()
}

func GenerateRandomID() int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(1000000) + 1
}

func (p *WorkerPoolAdapter) worker(workerID int) {
	defer func() {
		p.Wg.Done()
	}()

	for task := range p.Tasks {
		select {
		case <-task.Context.Done():
			log.Printf("[WORKER] client disconnected, cancelling task (%d)", task.Id)
			break
		default:
			log.Printf("[WORKER] (%d) running", task.Id)
			containerName := lib.GetContainerName(12)
			// Creating a blank dir
			// hostDir := fmt.Sprintf("%s%s", config.Data.Get("app.codingDirectory").(string), task.workingDirectory)
			hostDir := fmt.Sprintf("%s%s", config.Data.Get("app.codingDirectory").(string), containerName)

			coder := agent.New()

			coder = &agent.AgentAdapter{
				CoderAgent: types.CoderAgent{
					SystemPrompt:        task.SystemPrompt,
					UserPrompt:          task.UserPrompt,
					DockerImage:         task.DockerImage,
					DockerContainerName: containerName,
					LLMModel:            task.LLMModel,
					MaxRetry:            task.MaxRetry,
					WorkingDirectory:    hostDir,
					MaxTimeOut:          1,
					Logger:              task.Logger,
					Instrumentation:     types.InstrumentationStats{},
					Context:             task.Context,
					Cancel:              task.Cancel,
					Task:                &task,
				},
			}
			coder.StartTimer()
			coder.Run()
			coder.EndTimer()
		}
	}
}
