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

	"github.com/dev6699/cube/node"
	"github.com/dev6699/cube/queue"
	"github.com/dev6699/cube/scheduler"
	"github.com/dev6699/cube/store"
	"github.com/dev6699/cube/task"
	"github.com/dev6699/cube/worker"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

type Manager struct {
	Pending       queue.Queue[task.TaskEvent]
	TaskDb        store.Store[*task.Task]
	EventDb       store.Store[*task.TaskEvent]
	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
	LastWorker    int
	WorkerNodes   []*node.Node
	Scheduler     scheduler.Scheduler
}

func New(workers []string, schedulerType string, dbType string) (*Manager, error) {
	var nodes []*node.Node
	workerTaskMap := make(map[string][]uuid.UUID)
	for _, worker := range workers {
		workerTaskMap[worker] = []uuid.UUID{}

		nAPI := fmt.Sprintf("http://%s", worker)
		n := node.New(worker, nAPI, "worker")
		nodes = append(nodes, n)
	}

	var s scheduler.Scheduler
	switch schedulerType {
	case "roundrobin":
		s = &scheduler.RoundRobin{Name: "roundrobin"}

	case "epvm":
		s = &scheduler.Epvm{Name: "epvm"}

	default:
		s = &scheduler.RoundRobin{Name: "roundrobin"}
	}

	var ts store.Store[*task.Task]
	var es store.Store[*task.TaskEvent]
	switch dbType {
	case "memory":
		ts = store.NewInMemoryStore[*task.Task]()
		es = store.NewInMemoryStore[*task.TaskEvent]()

	case "bolt":
		var err error
		ts, err = store.NewBoltStore[*task.Task]("tasks.db", 0600, "tasks")
		if err != nil {
			return nil, err
		}
		es, err = store.NewBoltStore[*task.TaskEvent]("events.db", 0600, "events")
		if err != nil {
			return nil, err
		}
	}

	return &Manager{
		Pending:       queue.Queue[task.TaskEvent]{},
		Workers:       workers,
		TaskDb:        ts,
		EventDb:       es,
		TaskWorkerMap: make(map[uuid.UUID]string),
		WorkerTaskMap: workerTaskMap,
		LastWorker:    0,
		WorkerNodes:   nodes,
		Scheduler:     s,
	}, nil
}

func (m *Manager) AddTask(te task.TaskEvent) {
	m.Pending.Enqueue(te)
}

func (m *Manager) GetTasks() []*task.Task {
	tasks, err := m.TaskDb.List()
	if err != nil {
		return []*task.Task{}
	}

	return tasks
}

func (m *Manager) SelectWorker(t task.Task) (*node.Node, error) {
	candidates := m.Scheduler.SelectCandidateNodes(t, m.WorkerNodes)
	if candidates == nil {
		return nil, fmt.Errorf("[manager] no available candidate for task: %s", t.ID)
	}

	scores := m.Scheduler.Score(t, candidates)
	selectedNode := m.Scheduler.Pick(scores, candidates)
	return selectedNode, nil
}

func (m *Manager) SendWork() error {
	if m.Pending.Len() == 0 {
		return nil
	}

	te, ok := m.Pending.Dequeue()
	if !ok {
		return nil
	}

	err := m.EventDb.Put(te.ID.String(), &te)
	if err != nil {
		return err
	}

	t := te.Task
	taskWorker, ok := m.TaskWorkerMap[t.ID]
	if ok {
		persistedTask, err := m.TaskDb.Get(t.ID.String())
		if err != nil {
			return err
		}
		if te.State == task.Completed &&
			task.ValidStateTransiton(persistedTask.State, te.State) {
			return m.stopTask(taskWorker, t.ID.String())
		}

		log.Printf("[manager] invalid request: existing task is %s is in state %d and cannot transition to the completed state\n",
			persistedTask.ID.String(), persistedTask.State)
		return nil
	}

	w, err := m.SelectWorker(t)
	if err != nil {
		return err
	}

	m.WorkerTaskMap[w.Name] = append(m.WorkerTaskMap[w.Name], t.ID)
	m.TaskWorkerMap[t.ID] = w.Name

	t.State = task.Scheduled
	m.TaskDb.Put(t.ID.String(), &t)

	data, err := json.Marshal(te)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://%s/tasks", w.Name)
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

func (m *Manager) stopTask(worker string, taskID string) error {
	client := &http.Client{}
	url := fmt.Sprintf("http://%s/tasks/%s", worker, taskID)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

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
			key := t.ID.String()
			taskPersisted, err := m.TaskDb.Get(key)
			if err != nil {
				continue
			}

			taskPersisted.State = t.State
			taskPersisted.StartTime = t.StartTime
			taskPersisted.FinishTime = t.FinishTime
			taskPersisted.ContainerID = t.ContainerID
			taskPersisted.HostPorts = t.HostPorts

			m.TaskDb.Put(key, taskPersisted)
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
	m.TaskDb.Put(t.ID.String(), t)

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
