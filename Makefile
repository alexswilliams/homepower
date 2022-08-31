CGO_ENABLED=0

bin/main: deps $(shell find . -name '*.go')
	go build  -o bin/main -ldflags="-s -w" -tags netgo -a  cmd/main.go

deps: go.mod
	go mod download

run: bin/main
	./bin/main

clean:
	rm -rf bin vendor

docker-local:
	docker build -f build/package/Dockerfile .

.PHONY: deps run clean docker-local
