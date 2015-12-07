TEST?=./...
NAME = $(shell awk -F\" '/^const Name/ { print $$2 }' main.go)
VERSION = $(shell awk -F\" '/^const Version/ { print $$2 }' main.go)
DEPS = $(shell go list -f '{{range .TestImports}}{{.}} {{end}}' ./...)
CURRENT_DIR = $(shell pwd)

all: deps build

deps:
	@./install_consul.sh
	go get -d -v ./...
	echo $(DEPS) | xargs -n1 go get -d

updatedeps:
	go get -u -v ./...
	echo $(DEPS) | xargs -n1 go get -d

build: deps
	@mkdir -p bin/
	go build -o bin/$(NAME)

test: deps
	PATH=$(CURRENT_DIR)/bin:$(PATH) go test $(TEST) $(TESTARGS)
	go vet $(TEST)

xcompile: deps test
  # go get github.com/mitchellh/gox
	@rm -rf build/
	@mkdir -p build
	gox \
		-os="darwin" \
		-os="dragonfly" \
		-os="freebsd" \
		-os="linux" \
		-os="openbsd" \
		-os="solaris" \
		-os="windows" \
		-output "dist/$(NAME)_$(VERSION)_{{.OS}}_{{.Arch}}"

.PHONY: all deps updatedeps build test xcompile package
