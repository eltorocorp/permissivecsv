local: build test

prebuild:
	@echo Preparing build tooling...
	@go get -u github.com/eltorocorp/drygopher/drygopher
.PHONY: prebuild

build:
	@echo Updating dependencies...
	@go mod tidy -v
.PHONY: build

test:
	@drygopher -d -e "/mocks,/interfaces,/cmd,/host,'iface$$','drygopher$$','types$$'" -s 0
.PHONY: test