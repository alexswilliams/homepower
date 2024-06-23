CGO_ENABLED=0

bin/main: deps $(shell find . -name '*.go')
	go build  -o bin/main -ldflags="-s -w" -tags netgo -a  cmd/main.go

test: $(shell find . -name '*.go')
	go test ./device/...

deps: go.mod
	go mod download

run: bin/main
	HOMEPOWER_DEVICE_CONFIG_FILEPATH=config/exampleDeviceManifest.yaml HOMEPOWER_CREDENTIAL_FILEPATH=config/exampleCredentials.yaml ./bin/main

clean:
	rm -rf bin vendor

docker-local:
	docker build -f build/package/Dockerfile .

podman-local:
	podman build -f build/package/Dockerfile .

.PHONY: deps run clean docker-local podman-local test
