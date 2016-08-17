PACKAGES=$(shell go list ./... | grep -v /vendor/)
TESTARGS?=
CURRENT_DIR = $(shell pwd)
SOURCEDIR = $(CURRENT_DIR)
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')

all: deps build

deps:
	@./install_consul.sh

build: deps test
	@mkdir -p bin/
	go build -o bin/marathon-consul

test: deps $(SOURCES)
	PATH=$(CURRENT_DIR)/bin:$(PATH) go test $(PACKAGES) $(TESTARGS)
	go vet $(PACKAGES)

release:
	@rm -rf dist
	@go get github.com/laher/goxc
	goxc
	goxc bump
	git add .goxc.json
	git commit -m "Bumped version"

.PHONY: all build test xcompile package dist
