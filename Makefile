.PHONY: all
all: test

.PHONY: test
test:
	go test -v ./... -cover

.PHONY: bench
bench:
	go clean -testcache
	go test -v ./... -tags bench

.PHONY: report
report:
	go test ./... -coverprofile=./cover.out
	go tool cover -html=cover.out
