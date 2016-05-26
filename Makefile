.PHONY: check
check:
	go tool vet --all cmd monit util
	go tool vet --all *.go
	golint cmd
	golint monit
	golint util
	golint *.go

.PHONY: run
run:
	go run main.go start

.PHONY: format
format:
	goimports -w  cmd monit util
	goimports -w *.go
	gofmt -w  cmd monit util
	gofmt -w *.go
