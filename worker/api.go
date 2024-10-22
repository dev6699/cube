package worker

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Api struct {
	Address string
	Port    int
	Worker  *Worker
	Router  *chi.Mux
}

func NewApi(address string, port int, worker *Worker) *Api {
	return &Api{
		Address: address,
		Port:    port,
		Worker:  worker,
	}
}

func (a *Api) Start() error {
	a.initRouter()
	addr := fmt.Sprintf("%s:%d", a.Address, a.Port)
	return http.ListenAndServe(addr, a.Router)
}

func (a *Api) initRouter() {
	a.Router = chi.NewRouter()
	a.Router.Route("/tasks", func(r chi.Router) {
		r.Post("/", a.StartTaskHandler)
		r.Get("/", a.GetTasksHandler)
		r.Delete("/{taskID}", a.StopTaskHandler)
	})
	a.Router.Route("/stats", func(r chi.Router) {
		r.Get("/", a.GetStatsHandler)
	})
}
