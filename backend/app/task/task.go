package task

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"micro-tetzner-cloud-alarm/v2/app/config"
	bstore "micro-tetzner-cloud-alarm/v2/app/store"
	"net/http"
)

type Task struct {
	Type    string
	Owner   string
	Config  config.Config
	Context context.Context
	Store   bstore.Store
}

const (
	TASK_OWNER_WEB    = "web"
	TASK_OWNER_SYSTEM = "system"
)

func (t Task) Run() {
	httpClient := http.Client{}

	for _, service := range t.Config.Service {
		req, err := http.NewRequest("GET", service.URL+"/v1/servers", nil)
		if err != nil {
			log.Printf("[ERROR] %s", err)
			continue
		}

		req.Header.Set("Authorization", "Bearer "+service.Token)

		resp, err := httpClient.Do(req)
		if err != nil {
			log.Printf("[ERROR] %s", err)
			continue
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("[ERROR] %s", err)
			continue
		}

		//dataJson := bstore.JSON{}

		res := struct {
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
			Meta struct {
				Pagination struct {
					Page int `json:"page"`
				} `json:"pagination"`
			}
		}{}

		err = json.Unmarshal(body, &res)

		if err != nil {
			log.Printf("[ERROR] %s", err)
			continue
		}

		for _, server := range res.Servers {
			log.Printf("Server: %s", server.Name)
		}

		for _, server := range res.Servers {
			if t.Store.Get(service.Ident, server.PublicNet.Ipv4.Ip) != "" {
				log.Printf("Server %s already exists", server.Name)
				continue
			}

			msg := bstore.Message{
				Key:    server.PublicNet.Ipv4.Ip,
				Bucket: service.Ident,
				Type:   "server",
				Data:   server.Name,
			}

			t.Store.Save(&msg)
		}
	}
}
