GOBIN ?= ${GOPATH}/bin

all: cmd

fmt:
	go fmt ./...

vet:
	go vet ./...

cmd: fmt vet
	go build -ldflags="-w -s" -o bin/addon github.com/konveyor/tackle2-addon/cmd
