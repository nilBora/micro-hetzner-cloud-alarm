package main

import (

	//"encoding/json"
	//"io"
	//"micro-tetzner-cloud-alarm/v2/app/config"
	//bstore "micro-tetzner-cloud-alarm/v2/app/store"
	//"micro-tetzner-cloud-alarm/v2/app/task"

	//"net/http"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"micro-tetzner-cloud-alarm/v2/app/workflow"
	"net/http"
	"os"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/jessevdk/go-flags"
)

// CloudServers is the response structure for Hetzner API
type CloudServers struct {
	Servers []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"servers"`
}

type MyWorkflow struct {
}

type MyWorkflowInterface interface {
	FetchFromHetzner(task workflow.Task) CloudServers
}

type Options struct {
	Config      string        `short:"c" long:"config" env:"CONFIG" default:"config.yml" description:"config file"`
	Dbg         bool          `long:"dbg" env:"DEBUG" description:"show debug info"`
	Frequency   time.Duration `long:"frequency" env:"FREQUENCY" default:"10m" description:"task scheduler frequency in minutes"`
	StoragePath string        `short:"s" long:"storage_path" default:"/var/tmp/jtrw_hetzner_cloud.db" description:"Storage Path"`
}

var revision string

type JSON map[string]interface{}

func main() {
	log.Printf("[INFO] Micro HCA: %s\n", revision)

	var opts Options
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		log.Printf("[FATAL] %v", err)
		os.Exit(1)
	}

	callbacks := workflow.Callbacks{
		// "fetching": func() {
		// 	log.Printf("[INFO] Fetching data ...")
		// },
		"FetchFromHetzner": func(args ...interface{}) interface{} {
			task := args[0].(workflow.Task)
			log.Printf("[INFO] Fetching data from Hetzner ... %s", task.Name)

			return CloudServers{}
		},
	}

	myFw := workflow.UserWorkflow{}

	myFw.Callbacks = callbacks

	workflow.LoadWorkflow(myFw, callbacks)

	// cnf, err := config.LoadConfig(opts.Config)
	// if err != nil {
	// 	log.Printf("[FATAL] %v", err)
	// 	os.Exit(1)
	// }

	setupLog(opts.Dbg)

	// ctx, cancel := context.WithCancel(context.Background())
	// go func() {
	// 	sig := make(chan os.Signal, 1)
	// 	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	// 	<-sig
	// 	log.Printf("[WARN] Interrupted signal")
	// 	cancel()
	// }()

	// sec := bstore.Store{
	// 	StorePath: opts.StoragePath,
	// }

	// for _, transition := range cnf.Workflow.Transitions {
	// 	log.Printf("[INFO] %v", transition)
	// }

	// sec.JBolt = sec.NewStore()

	// httpClient := http.Client{}
	// for _, task := range cnf.Task {
	// 	if task.Type == "fetch" {
	// 		req, err := http.NewRequest("GET", task.Url, nil)
	// 		if err != nil {
	// 			log.Printf("[ERROR] %s", err)
	// 			continue
	// 		}
	// 		for _, header := range task.Headers {
	// 			log.Printf("[INFO] %s", header)
	// 			req.Header.Set(header.Name, header.Value)
	// 		}
	// 		//req.Header.Set("Authorization", "Bearer "+service.Token)

	// 		resp, err := httpClient.Do(req)
	// 		if err != nil {
	// 			log.Printf("[ERROR] %s", err)
	// 			continue
	// 		}

	// 		defer resp.Body.Close()

	// 		body, err := io.ReadAll(resp.Body)
	// 		if err != nil {
	// 			log.Printf("[ERROR] %s", err)
	// 			continue
	// 		}
	// 		response := JSON{}

	// 		json.Unmarshal(body, &response)

	// 		//log.Printf("[INFO] %s", body)

	// 	}
	// }

	// t := task.Task{
	// 	Owner:   task.TASK_OWNER_SYSTEM,
	// 	Config:  cnf,
	// 	Context: ctx,
	// 	Store:   sec,
	// }
	// t.Run()

	// for {
	// 	select {
	// 	case <-ctx.Done():
	// 		log.Printf("[WARN] Canceling current execution")
	// 		return
	// 	case <-time.After(opts.Frequency):
	// 		log.Printf("[INFO] Running task scheduler")
	// 		t := task.Task{
	// 			Owner:   task.TASK_OWNER_SYSTEM,
	// 			Config:  cnf,
	// 			Context: ctx,
	// 		}
	// 		t.Run()
	// 	}
	// }

}

func FetchFromHetzner(task workflow.Task) CloudServers {
	log.Printf("[INFO] Fetching data from Hetzner API...")

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

	log.Printf("[INFO] %d", cloudServers)

	//	wf.Data[task.ResponseStruct] = cloudServers
	//	wf.Data["fetchResult"] = cloudServers
	fmt.Println("Fetched data from Hetzner API.")

	return cloudServers
}

func (mw MyWorkflow) Ping() {
	log.Printf("[INFO] Pinging ...")
}

func setupLog(dbg bool) {
	if dbg {
		log.Setup(log.Debug, log.CallerFile, log.CallerFunc, log.Msec, log.LevelBraces)
		log.Printf("[DEBUG] debug mode ON")
		return
	}
	log.Setup(log.Msec, log.LevelBraces)
}
