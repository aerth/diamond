default: build lint 
all: default test bin/diamond-admin
lint:
	go vet ./...
test:
	go test ./...
build:
	env GOBIN=${PWD}/bin go install ./cmd/...
bin/diamond-admin: *.go ./cmd/diamond-admin/*.go
	@echo building diamond-admin client tool
	env GOBIN=${PWD}/bin go install ./cmd/diamond-admin
.PHONY += test all
