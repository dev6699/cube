export CUBE_MANAGER_HOST=localhost
export CUBE_MANAGER_PORT=5555
export CUBE_WORKER_HOST=localhost
export CUBE_WORKER_PORT=5556

.PHONY: run
run:
	go run main.go