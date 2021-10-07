ifeq ($(OS),Windows_NT)
    LIB_EXT := .dll
else
    LIB_EXT := .so
endif

LIB := librzpm${LIB_EXT}

all: rz-pm rzpm_c ${LIB}

.PHONY: tests integration-tests

integration-tests:
	go test -v -tags=integration ./...

tests:
	go test ./...

rz-pm: $(wildcard internal/**/*.go pkg/**/*.go main.go)
	go build

${LIB}: $(wildcard internal/**/*.go lib/*.go pkg/**/*.go)
	go build -o $@ -buildmode=c-shared ./lib

rzpm_c: c/rzpm.c ${LIB}
	${CC} -Wall -o $@ -I. -L. $< -lrzpm

clean:
	rm -f ${LIB} librzpm.h rzpm rzpm_c
