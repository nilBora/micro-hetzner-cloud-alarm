package workflow

import (
	"log"
	"micro-tetzner-cloud-alarm/v2/app/config"
)

type Workflow struct {
	Result    map[string]interface{}
	Callbacks map[string]func(args ...interface{}) interface{}
}

type Callbacks map[string]func(args ...interface{}) interface{}

var result interface{}
var results map[string]interface{}

func (wf *Workflow) Run(cnf config.Config) {
	log.Printf("[INFO] Running workflow\n")
	wf.Result = map[string]interface{}{}
	for _, stage := range cnf.Workflow.Stages {
		log.Printf("[INFO] Stage: %s\n", stage)
		task := getTask(cnf.Workflow.Tasks, stage)
		if wf.Callbacks[task.Func] != nil {
			log.Printf("[INFO] Task: %s\n", task.Name)
			if wf.Result[task.Store] != nil {
				store := wf.Result[task.Store]
				log.Printf("[INFO] Store: %v\n", store)
				wf.Result[task.Name] = wf.Callbacks[task.Func](task, store)
			} else {
				log.Printf("[INFO] Callback: %s\n", task.Func)
				wf.Result[task.Name] = wf.Callbacks[task.Func](task)
			}
		}
	}

	// for _, task := range cnf.Workflow.Tasks {
	// 	log.Printf("[INFO] Task: %s\n", task.Name)
	// }
}

func getTask(tasks []config.Task, name string) config.Task {
	for _, task := range tasks {
		if task.Name == name {
			return task
		}
	}
	return config.Task{}
}
