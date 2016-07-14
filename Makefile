.PHONY: install build doc fmt lint test watch godep

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
	ginkgo -r -slowSpecThreshold 60 -tags all

watch:
	ginkgo watch -r -tags all

godep:
	godep save $(go list ./... | ag -v /vendor/)
