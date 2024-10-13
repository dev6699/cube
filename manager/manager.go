package manager

import (
	"github.com/dev6699/cube/queue"
	"github.com/dev6699/cube/task"
	"github.com/google/uuid"
)

type Manager struct {
	Pending       queue.Queue[task.TaskEvent]
	TaskDb        map[string][]*task.Task
	EventDb       map[string][]*task.TaskEvent
	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
}

func (m *Manager) SelectWorker() {}

func (m *Manager) UpdateTasks() {}

func (m *Manager) SendWork() {}
