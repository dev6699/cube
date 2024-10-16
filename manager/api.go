package manager

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Api struct {
	Address string
	Port    int
	Manager *Manager
	Router  *chi.Mux
}

func NewApi(address string, port int, manager *Manager) *Api {
	return &Api{
		Address: address,
		Port:    port,
		Manager: manager,
	}
}

func (a *Api) Start() error {
	a.initRouter()
	addr := fmt.Sprintf("%s:%d", a.Address, a.Port)
	log.Println("[manager] listening on", addr)
	return http.ListenAndServe(addr, a.Router)
}

func (a *Api) initRouter() {
	a.Router = chi.NewRouter()
	a.Router.Route("/tasks", func(r chi.Router) {
		r.Post("/", a.StartTaskHandler)
		r.Get("/", a.GetTasksHandler)
		r.Delete("/{taskID}", a.StopTaskHandler)
	})
}
