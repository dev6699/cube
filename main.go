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

	ctx := context.Background()
	w := worker.New("worker-1")

	wapi := worker.NewApi(whost, wport, w)
	go w.RunTasks(ctx)
	go w.CollectStats(ctx)
	go wapi.Start()

	workers := []string{fmt.Sprintf("%s:%d", whost, wport)}
	m := manager.New(workers)
	go m.ProcessTasks(ctx)
	go m.UpdateTasks(ctx)

	mapi := manager.NewApi(mhost, mport, m)
	return mapi.Start()
}
