.PHONY: check
check:
	go tool vet --all agent cmd data util
	go tool vet --all *.go
	golint cmd
	golint data
	golint agent
	golint util
	golint *.go

.PHONY: build
build:
	go build main.go

.PHONY: run
run:
	go run main.go start

.PHONY: install
install: build
	intall cmonit /cmonit/

.PHONY: format
format:
	goimports -w  agent cmd data util
	goimports -w *.go
	gofmt -w  agent cmd data util
	gofmt -w *.go
