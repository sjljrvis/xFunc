package main

import (
	"codexec/config"
	"codexec/rpc"
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

func main() {
	godotenv.Load()
	config.Load()
	go rpc.StartRPCServer()
	select {}
}
