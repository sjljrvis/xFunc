package workflow

import (
	"fmt"
	"log"
)

type Action struct {
	Name        string
	Description string
}

type Workflow struct {
	Id      string
	Status  string
	Actions []Action
}

func (workflow Workflow) Run() {
	log.Println("Running flow")
	fmt.Println(workflow)
}
