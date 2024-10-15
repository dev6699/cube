package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dev6699/cube/manager"
	"github.com/dev6699/cube/task"
	"github.com/dev6699/cube/worker"
	"github.com/google/uuid"
)

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}

func run() error {
	ctx := context.Background()
	w := worker.New("worker-1")

	host := "127.0.0.1"
	port := 5555
	api := worker.NewApi(host, port, w)

	go runTasks(ctx, w)
	go w.CollectStats()
	go api.Start()

	workers := []string{fmt.Sprintf("%s:%d", host, port)}
	m := manager.New(workers)

	for i := 0; i < 3; i++ {
		t := task.Task{
			ID:    uuid.New(),
			Name:  fmt.Sprintf("test-container-%d", i),
			State: task.Scheduled,
			Image: "postgres:12-alpine",
			Env:   []string{"POSTGRES_USER=cube", "POSTGRES_PASSWORD=secret"},
		}
		te := task.TaskEvent{
			ID:    uuid.New(),
			State: task.Running,
			Task:  t,
		}
		m.AddTask(te)
		err := m.SendWork()
		if err != nil {
			log.Println(err)
		}
	}

	go func() {
		for {
			log.Println("Update tasks")
			err := m.UpdateTasks()
			if err != nil {
				log.Println(err)
			}
			time.Sleep(15 * time.Second)
		}
	}()

	for {
		log.Println("Tasks state:")
		for _, t := range m.TaskDb {
			fmt.Printf("Task id: %s, state: %d\n", t.ID, t.State)
		}
		time.Sleep(15 * time.Second)
	}
}

func runTasks(ctx context.Context, w *worker.Worker) {
	for {
		if w.Queue.Len() != 0 {
			_, err := w.RunTask(ctx)
			if err != nil {
				log.Printf("Error running task: %v\n", err)
			}
		} else {
			// log.Printf("No tasks to process currently.\n")
		}
		// log.Println("Sleeping for 10 seconds.")
		time.Sleep(10 * time.Second)
	}
}
