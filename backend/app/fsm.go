package main

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
	URL            string   `yaml:"url,omitempty"`
	Method         string   `yaml:"method,omitempty"`
	Headers        []Header `yaml:"headers,omitempty"`
	ResponseStruct string   `yaml:"responseStruct,omitempty"`
	Store          string   `yaml:"store,omitempty"`
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
func (wf *WorkflowFSM) FetchFromHetzner(task Task) {
	log.Println("Fetching data from Hetzner API...")
	client := &http.Client{}
	req, err := http.NewRequest(task.Method, task.URL, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, header := range task.Headers {
		req.Header.Add(header.Name, header.Value)
	}

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	cloudServers := CloudServers{}
	if err := json.Unmarshal(body, &cloudServers); err != nil {
		fmt.Println("Error unmarshalling response:", err)
		return
	}

	log.Println(cloudServers)

	wf.Data[task.ResponseStruct] = cloudServers
	fmt.Println("Fetched data from Hetzner API.")
}

// SaveInStore saves the data to a store (for example, a file)
func (wf *WorkflowFSM) SaveInStore(task Task) {
	log.Printf("[INFO] Saving data ...")
	data, ok := wf.Data[task.ResponseStruct]
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

func main() {
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
	log.Printf("[INFO] Worflow: %v", config)
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
			"fetch_from_hetzner": func(ctx context.Context, e *fsm.Event) {
				task := getTaskByName(workflow.Tasks, "fetch_from_hetzner")
				workflowFSM.FetchFromHetzner(task)
				workflowFSM.StateMachine.Event(ctx, "save_in_store")
			},
			"save_in_store": func(_ context.Context, e *fsm.Event) {
				task := getTaskByName(workflow.Tasks, "save_in_store")
				workflowFSM.SaveInStore(task)
			},
		},
	)

	// Step 3: Run the FSM
	// The FSM will transition from start -> fetch_from_hetzner -> save_in_store
	// The data fetched from Hetzner API will be saved in a file
	// The file will be named as the store specified in the task
	// The data will be saved in JSON format
	// The data will be saved in the same directory as the binary
	err = workflowFSM.StateMachine.Event(ctx, "start")
	if err != nil {
		fmt.Println("Error running FSM:", err)
		return
	}

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
