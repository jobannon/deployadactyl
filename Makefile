.PHONY: install build doc fmt lint test watch godep server

install:
	go get -t -v ./...

build:
	go build

doc:
	godoc -http=:6060

fmt:
	go fmt ./...

lint:
	golint ./...

test:
	ginkgo -r

watch:
	ginkgo watch -r

godep:
	godep save $(go list ./... | grep /vendor/)

server:
	go run server.go
