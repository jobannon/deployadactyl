.PHONY: build dependencies doc fmt lint server test watch

build:
	go build

dependencies:
	git submodule update --init --recursive

doc:
	godoc -http=:6060

fmt:
	for package in $$(go list ./... | grep -v /vendor/); do go fmt $$package; done

lint:
	for package in $$(go list ./... | grep -v /vendor/); do golint $$package; done

server:
	go run server.go

test:
	ginkgo -r

watch:
	ginkgo watch -r
