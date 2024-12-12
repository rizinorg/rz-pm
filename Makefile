all: rz-pm

.PHONY: clean tests integration-tests update-deps

integration-tests:
	go test -v -tags=integration ./...

tests:
	go test ./...

rz-pm: $(wildcard internal/**/*.go pkg/**/*.go main.go)
	go build

clean:
	rm -f rz-pm

update-deps:
	go get -u ./...
	go mod tidy
