package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dev6699/cube/task"
	"github.com/docker/docker/client"
)

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}

func run() error {
	fmt.Println("create a test container")

	dc, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	c := task.Config{
		Name:  "test-container-1",
		Image: "postgres:12-alpine",
		Env: []string{
			"POSTGRES_USER=cube",
			"POSTGRES_PASSWORD=secret",
		},
	}

	d := task.Docker{
		Client: dc,
		Config: c,
	}

	ctx := context.Background()
	result, err := d.Run(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("Container %s is running with config %v\n", result.ContainerId, c)

	time.Sleep(5 * time.Second)

	result, err = d.Stop(ctx, result.ContainerId)
	if err != nil {
		return err
	}

	fmt.Printf("Container %s has been stopped and removed\n", result.ContainerId)
	return nil
}
