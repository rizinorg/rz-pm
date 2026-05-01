all: rz-pm

.PHONY: clean tests integration-tests update-deps

integration-tests:
	go test -v -tags=integration ./...

tests:
	go test ./...

rz-pm: $(shell find pkg internal -type f -name '*.go' 2>/dev/null) main.go go.mod go.sum
	go build

clean:
	rm -f rz-pm

update-deps:
	go get -u ./...
	go mod tidy
	go mod vendor
