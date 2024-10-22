.PHONY: manager
manager:
	go run main.go manager -w "localhost:5556,localhost:5557,localhost:5558"

.PHONY: worker-1
worker-1:
	go run main.go worker

.PHONY: worker-2
worker-2:
	go run main.go worker -p 5557

.PHONY: worker-3
worker-3:
	go run main.go worker -p 5558

.PHONY: node
node:
	go run main.go node

.PHONY: status
status:
	go run main.go status

.PHONY: run
run:
	go run main.go run

ID=266592cd-960d-4091-981c-8c25c44b1018

.PHONY: stop
stop:
	go run main.go stop ${ID}