package main

import (
	"codexec/cli"
	"codexec/config"
	"codexec/rpc"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
)

const (
	TimeFormat = "02/02/2006 3:04:05 pm"
)

func init() {
	log.SetFlags(0)
	log.SetPrefix(time.Now().Format(time.RFC3339) + " [INFO] : ")
}

// mainProcess is the core logic of the application.
func mainProcess() {
	godotenv.Load()
	config.Load()
	go rpc.StartRPCServer()
	select {}
}

func main() {
	background := flag.Bool("background", false, "Run as a background service")
	forked := flag.Bool("background-forked", false, "Internal flag to indicate background mode")
	viewLog := flag.Bool("logs", false, "View logs from the log file in real-time (like tail -f)")
	flag.Parse()
	fmt.Sprintln(background, forked, viewLog)

	if *viewLog {
		if err := cli.TailLogs(); err != nil {
			log.Println("Error displaying logs:", err)
		}
		return
	} else {
		if *background {
			if err := cli.RunInBackground(); err != nil {
				log.Println("Failed to start background service:", err)
			}
			return
		}

		if *forked {
			cli.SetupLogging()
			mainProcess()
			return
		}

		log.Println("Running in foreground mode")
		mainProcess()
	}

}
