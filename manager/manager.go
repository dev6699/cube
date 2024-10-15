package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/dev6699/cube/queue"
	"github.com/dev6699/cube/task"
	"github.com/dev6699/cube/worker"
	"github.com/google/uuid"
)

type Manager struct {
	Pending       queue.Queue[task.TaskEvent]
	TaskDb        map[uuid.UUID]*task.Task
	EventDb       map[uuid.UUID]*task.TaskEvent
	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
	LastWorker    int
}

func New(workers []string) *Manager {
	workerTaskMap := make(map[string][]uuid.UUID)
	for _, worker := range workers {
		workerTaskMap[worker] = []uuid.UUID{}
	}

	return &Manager{
		Pending:       queue.Queue[task.TaskEvent]{},
		Workers:       workers,
		TaskDb:        make(map[uuid.UUID]*task.Task),
		EventDb:       make(map[uuid.UUID]*task.TaskEvent),
		TaskWorkerMap: make(map[uuid.UUID]string),
		WorkerTaskMap: workerTaskMap,
		LastWorker:    0,
	}
}

func (m *Manager) AddTask(te task.TaskEvent) {
	m.Pending.Enqueue(te)
}

func (m *Manager) SelectWorker() string {
	var newWorker int
	if m.LastWorker+1 < len(m.Workers) {
		newWorker = m.LastWorker + 1
		m.LastWorker++
	} else {
		newWorker = 0
		m.LastWorker = 0
	}

	return m.Workers[newWorker]
}

func (m *Manager) SendWork() error {
	if m.Pending.Len() == 0 {
		return nil
	}

	te, ok := m.Pending.Dequeue()
	if !ok {
		return nil
	}

	w := m.SelectWorker()
	t := te.Task
	m.EventDb[te.ID] = &te
	m.WorkerTaskMap[w] = append(m.WorkerTaskMap[w], t.ID)
	m.TaskWorkerMap[t.ID] = w

	t.State = task.Scheduled
	m.TaskDb[t.ID] = &t

	data, err := json.Marshal(te)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://%s/tasks", w)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		m.Pending.Enqueue(te)
		return err
	}

	d := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		var e worker.ErrResponse
		err := d.Decode(&e)
		if err != nil {
			return err
		}
	}

	var createdTask task.Task
	err = d.Decode(&createdTask)
	if err != nil {
		return err
	}
	log.Printf("task created %#v\n", t)
	return nil
}

func (m *Manager) UpdateTasks() error {
	for _, worker := range m.Workers {
		url := fmt.Sprintf("http://%s/tasks", worker)
		resp, err := http.Get(url)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("invalid status %v", resp.StatusCode)
		}

		d := json.NewDecoder(resp.Body)
		var tasks []*task.Task
		err = d.Decode(&tasks)
		if err != nil {
			return err
		}

		for _, t := range tasks {
			_, ok := m.TaskDb[t.ID]
			if !ok {
				continue
			}

			m.TaskDb[t.ID].State = t.State
			m.TaskDb[t.ID].StartTime = t.StartTime
			m.TaskDb[t.ID].FinishTime = t.FinishTime
			m.TaskDb[t.ID].ContainerID = t.ContainerID
		}
	}

	return nil
}
