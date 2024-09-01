package workflow

import (
	"log"
	"micro-tetzner-cloud-alarm/v2/app/config"
	"reflect"
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
			log.Printf("[INFO] Store Name: %s\n", task.Store)
			log.Printf("[INFO] RES: %v", wf.Result)
			if wf.Result[task.Store] != nil {
				store := wf.Result[task.Store]
				log.Printf("[INFO] Store: %v\n", store)
				wf.Result[task.Name] = wf.Callbacks[task.Func](task, store)
			} else {
				log.Printf("[INFO] Callback: %s\n", task.Func)
				res := wf.Callbacks[task.Func](task)
				writerType := reflect.TypeOf(res)
				log.Printf("[INFO] Type: %v\n", writerType)
				log.Printf("[INFO] Result: %v\n", res)
				//result = res
				//results = map[string]interface{}{}
				//results[task.Name] = result

				wf.Result[task.Name] = result
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
