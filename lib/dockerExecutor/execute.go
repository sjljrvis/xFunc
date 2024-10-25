package dockerexecutor

import (
	"bytes"
	"codexec/lib"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type DockerExecuteParams struct {
	ContainerName    string
	DockerImage      string
	WorkingDirectory string
	Context          context.Context
	Cancel           context.CancelFunc
}

type DockerExecuteResponse struct {
	ExitCode int
	Stdout   string
}

func Run(params DockerExecuteParams) DockerExecuteResponse {
	log.Println("[EXECUTOR] - spawning docker-container")
	var executeResponse DockerExecuteResponse
	ctx := params.Context

	select {
	case <-ctx.Done():
		log.Println("Got cancel request")
		executeResponse.ExitCode = 0
		executeResponse.Stdout = "canceled"
		return executeResponse

	default:

		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			log.Printf("Failed to create Docker client: %v", err)
		}

		containerVolumeDirectory := "/app"

		if err := os.MkdirAll(params.WorkingDirectory, os.ModePerm); err != nil {
			log.Fatal(err)
		}

		containerConfig := &container.Config{
			Image: params.DockerImage,
			Cmd:   []string{"tail", "-f", "/dev/null"},
		}

		containerHostConfig := &container.HostConfig{
			Mounts: []mount.Mount{{
				Type:   mount.TypeBind,
				Source: params.WorkingDirectory,
				Target: containerVolumeDirectory,
			},
			},
		}

		log.Printf("[EXECUTOR] - container creating")
		resp, err := cli.ContainerCreate(ctx, containerConfig, containerHostConfig, nil, nil, params.ContainerName)
		if err != nil {
			log.Printf("Error creating container: %v", err)
		}

		log.Printf("[EXECUTOR] - container starting")
		if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
			log.Printf("Error starting container: %v", err)
		}

		commands := lib.GenerateCommands(params.WorkingDirectory)

		logFileName := filepath.Join(params.WorkingDirectory, fmt.Sprintf("%s_output.log", params.ContainerName))
		logFile, err := os.Create(logFileName)

		for _, cmd := range commands {

			log.Printf("[EXECUTOR] - running docker cmd = %s", cmd)
			execConfig := types.ExecConfig{
				Cmd:          []string{"/bin/sh", "-c", cmd},
				AttachStdout: true,
				AttachStderr: true,
			}

			// Create the exec instance
			log.Printf("[EXECUTOR] - container exec ")
			execID, err := cli.ContainerExecCreate(ctx, resp.ID, execConfig)
			if err != nil {
				log.Printf("Error creating exec: %s\n", err)
				continue
			}

			// Start the exec instance
			log.Printf("[EXECUTOR] - container exec attach ")
			execResp, err := cli.ContainerExecAttach(ctx, execID.ID, types.ExecStartCheck{})
			if err != nil {
				log.Printf("Error starting exec: %s\n", err)
				continue
			}

			var buf bytes.Buffer
			_, err = io.Copy(&buf, execResp.Reader)

			defer execResp.Close()

			_, err = stdcopy.StdCopy(logFile, logFile, execResp.Reader)
			if err != nil {
				log.Printf("Error copying output: %s\n", err)
			}

			// Inspect the exec instance to get the exit code
			inspectResp, err := cli.ContainerExecInspect(ctx, execID.ID)
			if err != nil {
				log.Printf("Error inspecting exec: %s\n", err)
				continue
			}

			executeResponse.ExitCode = inspectResp.ExitCode
			executeResponse.Stdout = buf.String()

			if inspectResp.ExitCode != 0 {
				log.Printf("Command '%s' exited with status code: %d (log: %s)\n", cmd, inspectResp.ExitCode, logFileName)
				break
			}
		}

		log.Printf("[EXECUTOR] - container stop ")
		err = cli.ContainerStop(ctx, params.ContainerName, container.StopOptions{})
		if err != nil {
			log.Printf("Error stoping container: %v", err)
		}

		log.Printf("[EXECUTOR] - container remove ")
		err = cli.ContainerRemove(ctx, params.ContainerName, container.RemoveOptions{})
		if err != nil {
			log.Printf("Error creating container: %v", err)
		}
		return executeResponse
	}

}
