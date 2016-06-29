.PHONY: check
check:
	go tool vet --all agent cmd data util test
	go tool vet --all *.go
	golint cmd
	golint data
	golint agent
	golint util
	golint test
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
	goimports -w  agent cmd data util test
	goimports -w *.go
	gofmt -w  agent cmd data util test
	gofmt -w *.go

.PHONY: image
image:
	docker build -t yeasy/cmonit .
