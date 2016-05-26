.PHONY: check
check:
	go tool vet --all cmd database agent util
	go tool vet --all *.go
	golint cmd
	golint database
	golint agent
	golint util
	golint *.go

.PHONY: run
run:
	go run main.go start

.PHONY: format
format:
	goimports -w  cmd database agent util
	goimports -w *.go
	gofmt -w  cmd database agent util
	gofmt -w *.go
