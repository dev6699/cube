package main

import (
	"context"
	"log"
	"time"

	"github.com/dev6699/cube/worker"
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
	api := worker.NewApi("127.0.0.1", 5555, w)

	go runTasks(ctx, w)

	return api.Start()
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
