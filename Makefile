all: rz-pm

.PHONY: tests integration-tests

integration-tests:
	go test -v -tags=integration ./...

tests:
	go test ./...

rz-pm: $(wildcard internal/**/*.go pkg/**/*.go main.go)
	go build

clean:
	rm -f rz-pm
