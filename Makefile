.PHONY: all build test

all: build vet fmt

build: gtenlog.go
	go build

fmt: gtenlog.go
	go fmt

vet: gtenlog.go
	go vet

# test: gtenlog_test.go
# 	go test
