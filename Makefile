TEST?=./...
TESTARGS?=
DEPS = $(shell go list -f '{{range .TestImports}}{{.}} {{end}}' ./...)
CURRENT_DIR = $(shell pwd)
SOURCEDIR = $(CURRENT_DIR)
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')

all: deps build

deps:
	@./install_consul.sh
	go get -d -v ./...
	echo $(DEPS) | xargs -n1 go get -d

updatedeps:
	go get -u -v ./...
	echo $(DEPS) | xargs -n1 go get -d

build: deps test
	@mkdir -p bin/
	go build -o bin/marathon-consul

test: deps $(SOURCES)
	PATH=$(CURRENT_DIR)/bin:$(PATH) go test $(TEST) $(TESTARGS)
	go vet $(TEST)

release:
	@rm -rf dist
	@go get github.com/laher/goxc
	goxc
	goxc bump
	git add .goxc.json
	git commit -m "Bumped version"

.PHONY: all deps updatedeps build test xcompile package dist
