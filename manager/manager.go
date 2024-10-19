package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dev6699/cube/queue"
	"github.com/dev6699/cube/task"
	"github.com/dev6699/cube/worker"
	"github.com/docker/go-connections/nat"
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

func (m *Manager) GetTasks() []*task.Task {
	tasks := []*task.Task{}
	for _, t := range m.TaskDb {
		tasks = append(tasks, t)
	}
	return tasks
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
	log.Printf("[manager] task sent %#v\n", t)
	return nil
}

func (m *Manager) ProcessTasks(ctx context.Context) error {
	for {
		select {
		case <-time.NewTicker(10 * time.Second).C:
			log.Println("[manager] processing tasks:", m.Pending.Len())
			err := m.SendWork()
			if err != nil {
				log.Printf("[manager] error send task to worker: %v\n", err)
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func (m *Manager) UpdateTasks(ctx context.Context) error {
	for {
		select {
		case <-time.NewTicker(15 * time.Second).C:
			log.Println("[manager] updating tasks")
			err := m.updateTasks()
			if err != nil {
				log.Printf("[manager] error update task: %v\n", err)
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func (m *Manager) updateTasks() error {
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
			m.TaskDb[t.ID].HostPorts = t.HostPorts
		}
	}

	return nil
}

func (m *Manager) DoHealthChecks(ctx context.Context) error {
	for {
		select {
		case <-time.NewTicker(60 * time.Second).C:
			log.Println("[manager] checking health")
			m.doHealthChecks()

		case <-ctx.Done():
			return nil
		}
	}
}

func (m *Manager) doHealthChecks() {
	maxRestart := 3

	for _, t := range m.GetTasks() {
		if t.State == task.Running && t.RestartCount < maxRestart {
			err := m.checkTaskHealth(*t)
			if err != nil {
				log.Printf("[manager] error check task health: %v\n", err)
				if t.RestartCount < maxRestart {
					err = m.restartTask(t)
					if err != nil {
						log.Printf("[manager] error restart task: %v\n", err)
					}
				}
			}
		} else if t.State == task.Failed && t.RestartCount < maxRestart {
			log.Println("[manager] restarting failed task:", t.ID)
			err := m.restartTask(t)
			if err != nil {
				log.Printf("[manager] error restart task: %v\n", err)
			}
		}
	}
}

func (m *Manager) restartTask(t *task.Task) error {
	w := m.TaskWorkerMap[t.ID]
	t.State = task.Scheduled
	t.RestartCount++
	m.TaskDb[t.ID] = t

	te := task.TaskEvent{
		ID:        uuid.New(),
		State:     task.Running,
		Timestamp: time.Now(),
		Task:      *t,
	}
	data, err := json.Marshal(&te)
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

		return fmt.Errorf("failed to restart task; err = %v", e)
	}

	var newTask task.Task
	err = d.Decode(&newTask)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) checkTaskHealth(t task.Task) error {
	w := m.TaskWorkerMap[t.ID]
	hostPort := getHostPort(t.HostPorts)
	if hostPort == nil {
		return fmt.Errorf("invalid hostPort")
	}

	workerHost := strings.Split(w, ":")
	if len(workerHost) < 1 {
		return fmt.Errorf("invalid worker host")
	}

	url := fmt.Sprintf("http://%s:%s%s", workerHost[0], *hostPort, t.HealthCheck)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status code: %v", resp.StatusCode)
	}

	return nil
}

func getHostPort(ports nat.PortMap) *string {
	for _, pb := range ports {
		if len(pb) == 0 {
			continue
		}
		return &pb[0].HostPort
	}

	return nil
}
