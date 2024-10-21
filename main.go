package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/dev6699/cube/manager"
	"github.com/dev6699/cube/worker"
)

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}

func run() error {
	whost := os.Getenv("CUBE_WORKER_HOST")
	wport, _ := strconv.Atoi(os.Getenv("CUBE_WORKER_PORT"))
	mhost := os.Getenv("CUBE_MANAGER_HOST")
	mport, _ := strconv.Atoi(os.Getenv("CUBE_MANAGER_PORT"))
	dbType := "bolt" // "memory"

	ctx := context.Background()
	workers := []string{}
	workerCount := 3
	for i := 0; i < workerCount; i++ {
		w, err := worker.New(fmt.Sprintf("worker-%d", i+1), dbType)
		if err != nil {
			return err
		}
		port := wport + i
		workers = append(workers, fmt.Sprintf("%s:%d", whost, port))
		wapi := worker.NewApi(whost, port, w)
		go w.RunTasks(ctx)
		go w.CollectStats(ctx)
		go w.UpdateTasks(ctx)
		go wapi.Start()
	}

	m, err := manager.New(workers, "roundrobin", dbType)
	if err != nil {
		return err
	}
	go m.ProcessTasks(ctx)
	go m.UpdateTasks(ctx)
	go m.DoHealthChecks(ctx)

	mapi := manager.NewApi(mhost, mport, m)
	return mapi.Start()
}
