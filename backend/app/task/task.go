package task

import (
	"context"
	"micro-tracker-parser/v2/app/config"
	"micro-tracker-parser/v2/app/tracker"
)

type Task struct {
	Type      string
	Owner     string
	DateRange tracker.DateRange
	Config    config.Config
	Context   context.Context
}

const (
	TASK_OWNER_WEB    = "web"
	TASK_OWNER_SYSTEM = "system"
)

func (t Task) Run() {
	for _, service := range t.Config.Service {
		//do request

	}
}
