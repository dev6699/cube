package worker

import (
	"github.com/dev6699/cube/queue"
	"github.com/dev6699/cube/task"
	"github.com/google/uuid"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	Db        map[uuid.UUID]*task.Task
	TaskCount int
}

func (w *Worker) CollectStats() {}

func (w *Worker) RunTask() {}

func (w *Worker) StartTask() {}

func (w *Worker) StopTask() {}
