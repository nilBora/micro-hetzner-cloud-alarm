package main

import (

	//"encoding/json"
	//"io"
	"bytes"
	"context"
	"micro-tetzner-cloud-alarm/v2/app/config"
	bstore "micro-tetzner-cloud-alarm/v2/app/store"
	"os/signal"
	"syscall"

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
		Id        int    `json:"id"`
		Name      string `json:"name"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
		PublicNet struct {
			Ipv4 struct {
				Ip string `json:"ip"`
			} `json:"ipv4"`
		} `json:"public_net"`
	} `json:"servers"`
}

type Options struct {
	Config      string        `short:"c" long:"config" env:"CONFIG" default:"workflow.yml" description:"config file"`
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

	cnf, err := config.LoadConfig(opts.Config)
	if err != nil {
		log.Printf("[FATAL] %v", err)
		os.Exit(1)
	}

	setupLog(opts.Dbg)

	sec := bstore.Store{
		StorePath: opts.StoragePath,
	}

	sec.JBolt = sec.NewStore()

	callbacks := workflow.Callbacks{
		"fetchFromHetzner": func(args ...interface{}) interface{} {
			task := args[0].(config.Task)
			return fetchFromHetzner(task)
		},
		"checkInStore": func(args ...interface{}) interface{} {
			task := args[0].(config.Task)
			result := args[1].(CloudServers)
			log.Printf("[INFO] Checking in store %v", result)
			return checkInStore(task, result, sec)
		},
		"sendingToSlack": func(args ...interface{}) interface{} {
			task := args[0].(config.Task)

			if len(args) <= 1 || args[1] == nil {
				log.Printf("[ERROR] No data to send to slack")
				return nil
			}
			result := args[1].(CloudServers)
			log.Printf("[INFO] sendingToSlack %v, %v", task, result)

			return nil
		},
	}
	fw := workflow.Workflow{
		Callbacks: callbacks,
	}
	fw.Run(cnf)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Printf("[WARN] Interrupted signal")
		cancel()
	}()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[WARN] Canceling current execution")
			return
		case <-time.After(opts.Frequency):
			log.Printf("[INFO] Running task scheduler")
			fw.Run(cnf)
		}
	}
}

func fetchFromHetzner(task config.Task) CloudServers {
	log.Printf("[INFO] Fetching data from Hetzner API...")
	st := CloudServers{}

	client := &http.Client{}
	req, err := http.NewRequest(task.Method, task.Url, nil)
	if err != nil {
		fmt.Println(err)
		return st
	}

	for _, header := range task.Headers {
		req.Header.Add(header.Name, header.Value)
	}

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return st
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return st
	}

	if err := json.Unmarshal(body, &st); err != nil {
		fmt.Println("Error unmarshalling response:", err)
		return st
	}

	fmt.Println("Fetched data from Hetzner API.")

	return st
}

func checkInStore(task config.Task, servers CloudServers, sec bstore.Store) interface{} {
	saveServers := sec.Get("tasks", task.Name)

	if len(saveServers) > 0 {
		str, _ := json.Marshal(servers)
		if string(str) == saveServers {
			log.Printf("[INFO] No changes in store")
			return nil
		}

		storeServers := CloudServers{}
		if err := json.Unmarshal([]byte(saveServers), &storeServers); err != nil {
			log.Printf("[ERROR] %v", err)
			return nil

		}
		for _, server := range servers.Servers {
			for _, storeServer := range storeServers.Servers {
				if server.Id == storeServer.Id {
					servers.Servers = append(servers.Servers[:0], servers.Servers[1:]...)
				}
			}
		}

		log.Printf("[INFO] Changes detected in store")

		sec.Set("tasks", task.Name, string(str))

		return servers
	}

	str, _ := json.Marshal(servers)
	sec.Set("tasks", task.Name, string(str))

	return nil
}

func secdToSlack(url string, message string) {
	log.Printf("[INFO] Sending message to Slack")
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(message)))
	if err != nil {
		log.Printf("[ERROR] %v", err)
		return
	}

	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Printf("[ERROR] %v", err)
		return
	}
	defer res.Body.Close()

	log.Printf("[INFO] Message sent to Slack")
}

func setupLog(dbg bool) {
	if dbg {
		log.Setup(log.Debug, log.CallerFile, log.CallerFunc, log.Msec, log.LevelBraces)
		log.Printf("[DEBUG] debug mode ON")
		return
	}
	log.Setup(log.Msec, log.LevelBraces)
}
