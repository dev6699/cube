package worker

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dev6699/cube/task"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ErrResponse struct {
	HTTPStatusCode int
	Message        string
}

func (a *Api) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(a.Worker.GetTasks())
}

func (a *Api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	var te task.TaskEvent
	err := d.Decode(&te)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		e := ErrResponse{
			HTTPStatusCode: http.StatusBadRequest,
			Message:        fmt.Sprintf("Error unmarshalling body: %v\n", err),
		}
		json.NewEncoder(w).Encode(e)
		return
	}

	a.Worker.AddTask(te.Task)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(te.Task)
}

func (a *Api) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		w.WriteHeader(http.StatusBadRequest)
		e := ErrResponse{
			HTTPStatusCode: http.StatusBadRequest,
			Message:        "invalid taskID",
		}
		json.NewEncoder(w).Encode(e)
		return
	}

	tID, err := uuid.Parse(taskID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		e := ErrResponse{
			HTTPStatusCode: http.StatusBadRequest,
			Message:        fmt.Sprintf("Error parsing taskID: %v\n", err),
		}
		json.NewEncoder(w).Encode(e)
		return
	}

	taskToStop, err := a.Worker.Db.Get(tID.String())
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		e := ErrResponse{
			HTTPStatusCode: http.StatusNotFound,
			Message:        fmt.Sprintf("no task with ID %v found", tID),
		}
		json.NewEncoder(w).Encode(e)
		return
	}

	taskCopy := *taskToStop
	taskCopy.State = task.Completed
	a.Worker.AddTask(taskCopy)
	w.WriteHeader(http.StatusNoContent)
}

func (a *Api) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(a.Worker.Stats)
}
