package framework

import (
	"framework/pkg/auth"
	"framework/pkg/log"
	"framework/pkg/metric"
	"framework/pkg/recovery"
	"framework/pkg/server"
	"framework/pkg/trace"
	syslog "log"
)

type Application struct {
}

func NewApplication() *Application {
	return &Application{}
}

func (a *Application) Init() error {
	initList := make([]func() error, 0)
	initList = append(initList, auth.Init)
	initList = append(initList, log.Init)
	initList = append(initList, metric.Init)
	initList = append(initList, recovery.Init)
	initList = append(initList, trace.Init)
	initList = append(initList, server.Init)

	for _, f := range initList {
		if err := f(); err != nil {
			syslog.Fatalf("init failed err: %s\n", err.Error())
		}
	}

	return nil
}

func (a *Application) Run() error {
	server.Run()

	return nil
}
