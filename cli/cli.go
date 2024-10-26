package cli

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

const logFilePath = "app.log"

// setupLogging configures the logger to write to the log file.
func SetupLogging() {
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Failed to open log file:", err)
		return
	}
	log.SetOutput(logFile)
}

// runInBackground starts the application as a detached background process.
func RunInBackground() error {
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	cmd := exec.Command(os.Args[0], "--background-forked=true")
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting in background: %w", err)
	}

	fmt.Printf("Process started in background with PID: %d\n", cmd.Process.Pid)
	return nil
}

// tailLogs continuously prints new lines from the log file, mimicking `tail -f`.
func TailLogs() error {
	logFile, err := os.Open(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	_, err = logFile.Seek(0, os.SEEK_CUR)
	if err != nil {
		return fmt.Errorf("failed to seek to end of log file: %w", err)
	}

	buffer := make([]byte, 1024)
	for {
		n, err := logFile.Read(buffer)
		if err != nil && err.Error() != "EOF" {
			return fmt.Errorf("error reading log file: %w", err)
		}

		if n > 0 {
			fmt.Print(string(buffer[:n]))
		}

		time.Sleep(100 * time.Millisecond)
	}
}
