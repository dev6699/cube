package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/dev6699/cube/queue"
	"github.com/dev6699/cube/task"
	"github.com/google/uuid"
)

type Worker struct {
	Name      string
	Queue     queue.Queue[task.Task]
	Db        map[uuid.UUID]*task.Task
	TaskCount int
	Stats     *Stats
}

func New(name string) *Worker {
	return &Worker{
		Name:  name,
		Queue: queue.Queue[task.Task]{},
		Db:    make(map[uuid.UUID]*task.Task),
	}
}

func (w *Worker) CollectStats() {
	for {
		w.Stats = GetStats()
		w.Stats.TaskCount = w.TaskCount
		time.Sleep(15 * time.Second)
	}
}

func (w *Worker) RunTask(ctx context.Context) (*task.DockerResult, error) {
	taskQueued, ok := w.Queue.Dequeue()
	if !ok {
		return nil, nil
	}

	taskPersisted := w.Db[taskQueued.ID]
	if taskPersisted == nil {
		taskPersisted = &taskQueued
		w.Db[taskQueued.ID] = &taskQueued
	}

	var result *task.DockerResult
	var err error
	if task.ValidStateTransiton(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			result, err = w.StartTask(ctx, taskQueued)

		case task.Completed:
			result, err = w.StopTask(ctx, taskQueued)

		default:
			return nil, fmt.Errorf("invalid task state")
		}
	} else {
		return nil, fmt.Errorf("invalid state transition")
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
		t.State = task.Failed
		w.Db[t.ID] = &t
		return result, nil
	}

	t.ContainerID = result.ContainerId
	t.State = task.Running
	w.Db[t.ID] = &t

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
	w.Db[t.ID] = &t

	return result, nil
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) GetTasks() []*task.Task {
	tasks := []*task.Task{}
	for _, t := range w.Db {
		tasks = append(tasks, t)
	}
	return tasks
}
