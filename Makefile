dev:
	docker compose build
	docker compose up --abort-on-container-exit

test: fmt
	go test ./...

fmt:
	go fmt ./...

LINTER := $(shell command -v $(shell go env GOPATH)/bin/golangci-lint 2> /dev/null)
lint:
ifndef LINTER
	go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.40.1
endif
	$(shell go env GOPATH)/bin/golangci-lint run ./...
