.PHONY: all
all: test

.PHONY: test
test:
	go clean -testcache
	go test -v ./... -cover

.PHONY: report
report:
	go test ./... -coverprofile=./cover.out
	go tool cover -html=cover.out
