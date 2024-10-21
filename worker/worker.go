package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/dev6699/cube/queue"
	"github.com/dev6699/cube/stats"
	"github.com/dev6699/cube/store"
	"github.com/dev6699/cube/task"
	"github.com/docker/docker/api/types"
)

type Worker struct {
	Name      string
	Queue     queue.Queue[task.Task]
	Db        store.Store[*task.Task]
	TaskCount int
	Stats     *stats.Stats
}

func New(name string, taskDbType string) (*Worker, error) {

	var s store.Store[*task.Task]
	switch taskDbType {
	case "memory":
		s = store.NewInMemoryStore[*task.Task]()

	case "bolt":
		file := fmt.Sprintf("%s_tasks.db", name)
		var err error
		s, err = store.NewBoltStore[*task.Task](file, 0600, "tasks")
		if err != nil {
			return nil, err
		}
	}

	return &Worker{
		Name:  name,
		Queue: queue.Queue[task.Task]{},
		Db:    s,
	}, nil
}

func (w *Worker) CollectStats(ctx context.Context) error {
	for {
		select {
		case <-time.NewTicker(15 * time.Second).C:
			log.Println("[worker] collecting stats")
			w.Stats = stats.GetStats()
			w.Stats.TaskCount = w.TaskCount

		case <-ctx.Done():
			return nil
		}
	}
}

func (w *Worker) RunTasks(ctx context.Context) error {
	for {
		select {
		case <-time.NewTicker(10 * time.Second).C:
			if w.Queue.Len() != 0 {
				log.Println("[worker] running tasks")
				_, err := w.runTask(ctx)
				if err != nil {
					log.Printf("[worker] error running task: %v\n", err)
				}
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func (w *Worker) runTask(ctx context.Context) (*task.DockerResult, error) {
	taskQueued, ok := w.Queue.Dequeue()
	if !ok {
		return nil, nil
	}

	key := taskQueued.ID.String()
	taskPersisted, err := w.Db.Get(key)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			taskPersisted = &taskQueued
			err = w.Db.Put(key, &taskQueued)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	var result *task.DockerResult
	if task.ValidStateTransiton(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			result, err = w.StartTask(ctx, taskQueued)

		case task.Completed:
			result, err = w.StopTask(ctx, taskQueued)

		default:
			return nil, fmt.Errorf("invalid task state %v", taskQueued.State)
		}
	} else {
		return nil, fmt.Errorf("invalid state transition from %v to %v", taskPersisted.State, taskQueued.State)
	}

	return result, err
}

func (w *Worker) StartTask(ctx context.Context, t task.Task) (*task.DockerResult, error) {
	t.StartTime = time.Now().UTC()
	config := task.NewConfig(&t)
	d, err := task.NewDocker(config)
	if err != nil {
		return nil, err
	}

	result, err := d.Run(ctx)
	if err != nil {
		log.Printf("[worker] failed to start task: %#v\n", err)
		t.State = task.Failed
		w.Db.Put(t.ID.String(), &t)
		return result, nil
	}

	t.ContainerID = result.ContainerId
	t.State = task.Running
	err = w.Db.Put(t.ID.String(), &t)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (w *Worker) StopTask(ctx context.Context, t task.Task) (*task.DockerResult, error) {
	config := task.NewConfig(&t)
	d, err := task.NewDocker(config)
	if err != nil {
		return nil, err
	}

	result, err := d.Stop(ctx, t.ContainerID)
	if err != nil {
		return nil, err
	}

	t.FinishTime = time.Now().UTC()
	t.State = task.Completed
	err = w.Db.Put(t.ID.String(), &t)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) GetTasks() []*task.Task {
	tasks, err := w.Db.List()
	if err != nil {
		return []*task.Task{}
	}
	return tasks
}

func (w *Worker) UpdateTasks(ctx context.Context) error {
	for {
		select {
		case <-time.NewTicker(15 * time.Second).C:
			log.Println("[worker] updating tasks")
			err := w.updateTasks(ctx)
			if err != nil {
				log.Printf("[worker] failed to update tasks: %#v\n", err)
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func (w *Worker) updateTasks(ctx context.Context) error {
	tasks, err := w.Db.List()
	if err != nil {
		return err
	}

	for _, t := range tasks {
		if t.State != task.Running {
			continue
		}

		key := t.ID.String()
		resp, err := w.InspectTask(ctx, *t)
		if err != nil {
			log.Printf("[worker] failed to inspect task: %#v\n", err)
			t.State = task.Failed
			w.Db.Put(key, t)
			continue
		}

		if resp.State.Status == "exited" {
			t.State = task.Failed
			w.Db.Put(key, t)
			continue
		}

		t.HostPorts = resp.NetworkSettings.Ports
		w.Db.Put(key, t)
	}

	return nil
}

func (w *Worker) InspectTask(ctx context.Context, t task.Task) (types.ContainerJSON, error) {
	config := task.NewConfig(&t)
	d, err := task.NewDocker(config)
	if err != nil {
		return types.ContainerJSON{}, err
	}
	return d.Inspect(ctx, t.ContainerID)
}
