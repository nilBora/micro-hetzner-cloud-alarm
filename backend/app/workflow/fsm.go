package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/looplab/fsm"
)

type Callbacks map[string]func(args ...interface{}) interface{}

type UserWorkflow struct {
	Callbacks Callbacks
}

type Config struct {
	Workflow Workflow `yaml:"workflow"`
}

// Workflow represents the structure of the YAML file
type Workflow struct {
	Transitions []Transition `yaml:"transitions"`
	Tasks       []Task       `yaml:"tasks"`
}

type Transition struct {
	Name string `yaml:"name"`
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

type Task struct {
	Name           string   `yaml:"name"`
	Type           string   `yaml:"type"`
	Event          string   `yaml:"event"`
	Func           string   `yaml:"func"`
	URL            string   `yaml:"url,omitempty"`
	Method         string   `yaml:"method,omitempty"`
	Headers        []Header `yaml:"headers,omitempty"`
	ResponseStruct string   `yaml:"responseStruct,omitempty"`
	Store          string   `yaml:"store,omitempty"`
}

type Response struct {
	Key  string
	Data string
}

type Header struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// CloudServers is the response structure for Hetzner API
type CloudServers struct {
	Servers []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"servers"`
}

// WorkflowFSM struct
type WorkflowFSM struct {
	StateMachine *fsm.FSM
	Workflow     Workflow
	Data         map[string]interface{}
}

// FetchFromHetzner fetches data from Hetzner API
func (wf *WorkflowFSM) FetchFromHetzner(task Task) CloudServers {
	log.Println("Fetching data from Hetzner API...")

	cloudServers := CloudServers{}

	client := &http.Client{}
	req, err := http.NewRequest(task.Method, task.URL, nil)
	if err != nil {
		fmt.Println(err)
		return cloudServers
	}

	for _, header := range task.Headers {
		req.Header.Add(header.Name, header.Value)
	}

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return cloudServers
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return cloudServers
	}

	if err := json.Unmarshal(body, &cloudServers); err != nil {
		fmt.Println("Error unmarshalling response:", err)
		return cloudServers
	}

	log.Println(cloudServers)

	wf.Data[task.ResponseStruct] = cloudServers
	wf.Data["fetchResult"] = cloudServers
	fmt.Println("Fetched data from Hetzner API.")

	return cloudServers
}

// SaveInStore saves the data to a store (for example, a file)
func (wf *WorkflowFSM) SaveInStore(task Task, response interface{}) {
	log.Printf("[INFO] Saving data ...")
	log.Printf("[INFO] Prev Response: %v", response)
	data, ok := wf.Data["fetchResult"]
	if !ok {
		fmt.Println("No data found to save")
		return
	}

	file, err := os.Create(task.Store + ".json")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	dataBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling data:", err)
		return
	}

	file.Write(dataBytes)
	fmt.Println("Data saved to", task.Store+".json")
}

func (wf *WorkflowFSM) CheckResponse() {
	log.Println("Checking response...")
	data, ok := wf.Data["fetchResult"]
	if !ok {
		fmt.Println("No data found to check")
		return
	}

	fmt.Println("Data:", data)
}

func LoadWorkflow(wf UserWorkflow, callbacks Callbacks) {
	ctx := context.Background()
	// Step 1: Read and parse the YAML file
	yamlFile, err := ioutil.ReadFile("workflow.yml")
	if err != nil {
		fmt.Printf("Error reading YAML file: %v\n", err)
		return
	}

	var workflow Workflow
	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		fmt.Printf("Error unmarshalling YAML file: %v\n", err)
		return
	}
	//log.Printf("[INFO] Worflow: %v", config)
	workflow = config.Workflow
	workflowFSM := &WorkflowFSM{
		Workflow: workflow,
		Data:     make(map[string]interface{}),
	}

	// Step 2: Initialize the FSM
	fsmEvents := []fsm.EventDesc{}
	for _, t := range workflow.Transitions {
		fsmEvents = append(fsmEvents, fsm.EventDesc{
			Name: t.Name,
			Src:  []string{t.From},
			Dst:  t.To,
		})
	}

	workflowFSM.StateMachine = fsm.NewFSM(
		"start",
		fsmEvents,
		fsm.Callbacks{
			"fetching": func(ctx context.Context, e *fsm.Event) {
				log.Printf("In Callback fetch_from_hetzner...\n")
				log.Printf("Current state: %v\n", e.FSM.Current())
				task := getTaskByName(workflow.Tasks, e.FSM.Current())
				rersult := workflowFSM.FetchFromHetzner(task)
				fmt.Println("after_scan: " + e.FSM.Current())

				//wf.Callbacks[task.Func](task, ctx, e)

				err := workflowFSM.StateMachine.Event(ctx, task.Event, rersult)
				if err != nil {
					fmt.Println("Error running FSM:", err)
				}
				log.Printf("END Callback fetch_from_hetzner\n")
			},
			"after_fetching": func(ctx context.Context, e *fsm.Event) {

			},
			"checking": func(ctx context.Context, e *fsm.Event) {
				log.Printf("In Callback check_response...\n")
				log.Printf("Current state: %v\n", e.FSM.Current())
				//task := getTaskByName(workflow.Tasks, e.FSM.Current())
				response := e.Args[0]
				workflowFSM.CheckResponse()

				err := workflowFSM.StateMachine.Event(ctx, "check", response)
				if err != nil {
					fmt.Println("Error running FSM:", err)
				}

				log.Printf("END Callback check_response\n")
			},
			"saving": func(_ context.Context, e *fsm.Event) {
				response := e.Args[0]
				log.Printf("In Callback save_in_store...\n")
				task := getTaskByName(workflow.Tasks, e.FSM.Current())
				workflowFSM.SaveInStore(task, response)
			},
		},
	)

	// Step 3: Run the FSM

	fmt.Println("1:" + workflowFSM.StateMachine.Current())
	err = workflowFSM.StateMachine.Event(ctx, "run")
	if err != nil {
		fmt.Println("Error running FSM:", err)
		return
	}

	fmt.Println("2:" + workflowFSM.StateMachine.Current())

	// err = workflowFSM.StateMachine.Event(ctx, "fetch")
	// if err != nil {
	// 	fmt.Println("Error running FSM:", err)
	// 	return
	// }

	fmt.Println("Workflow completed successfully.")
}

// getTaskByName retrieves a task by its name from a list of tasks
func getTaskByName(tasks []Task, name string) Task {
	for _, task := range tasks {
		if task.Name == name {
			return task
		}
	}
	return Task{}
}
