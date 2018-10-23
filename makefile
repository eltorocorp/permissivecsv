local: build test

prebuild:
	@echo Preparing build tooling...
	@go get -u github.com/golang/dep/cmd/dep
.PHONY: prebuild

build:
	@echo Updating dependencies...
	@dep ensure
.PHONY: build

test:
	@go test ./...
.PHONY: test