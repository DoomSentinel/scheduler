default: build

run:
	docker-compose up -d --build

install:
	@go get github.com/golang/mock/mockgen@v1.4.3
	@go mod tidy

build: install generate
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o application main.go

generate:
	@go generate ./...

test: install
	@go test -cover ./...

.PHONY: test build install generate run