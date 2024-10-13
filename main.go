package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dev6699/cube/task"
	"github.com/dev6699/cube/worker"
)

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}

func run() error {
	w := worker.New("worker-1")
	t := task.NewTask(
		"test-container-1",
		"postgres:12-alpine",
		[]string{
			"POSTGRES_USER=cube",
			"POSTGRES_PASSWORD=secret",
		},
	)
	fmt.Println("starting task")
	w.AddTask(*t)

	ctx := context.Background()
	result, err := w.RunTask(ctx)
	if err != nil {
		return nil
	}

	t.ContainerID = result.ContainerId
	fmt.Printf("task %s is running in container %s\n", t.ID, t.ContainerID)
	time.Sleep(50 * time.Second)

	fmt.Printf("stopping task %s\n", t.ID)
	t.State = task.Completed
	w.AddTask(*t)
	result, err = w.RunTask(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("Container %s has been stopped and removed\n", result.ContainerId)
	return nil
}
