all: bin/diamond-admin test
test:
	go vet ./...
bin/diamond-admin:
	env GOBIN=${PWD}/bin go install ./cmd/diamond-admin
