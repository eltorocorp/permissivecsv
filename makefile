local: build test

prebuild:
	@echo Preparing build tooling...
	@go get -u github.com/golang/dep/cmd/dep
	@go install github.com/eltorocorp/drygopher
.PHONY: prebuild

build:
	@echo Updating dependencies...
	@dep ensure
.PHONY: build

test:
	@drygopher -d -e "/mocks,/interfaces,/cmd,/host,'iface$$','drygopher$$','types$$'" -s 0
.PHONY: test