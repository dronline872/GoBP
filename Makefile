PHONY: test
test:
	go test -race ./...

PHONY: run
run:
	go run cmd/
