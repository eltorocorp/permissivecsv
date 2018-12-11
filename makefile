local: build test

prebuild:
	@echo Preparing build tooling...
	@go get -u github.com/eltorocorp/drygopher/drygopher
.PHONY: prebuild

build:
	@echo Updating dependencies...
	@GO111MODULE=on go mod tidy -v
.PHONY: build

test:
	@GO111MODULE=on drygopher -d -e "/mocks,/interfaces,/cmd,/host,'iface$$','drygopher$$','types$$'" -s 0
.PHONY: test