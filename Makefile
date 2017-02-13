.DEFAULT_GOAL := build

PACKAGES = $(shell go list ./... | grep -v /vendor/)

TESTARGS ?= -race

CURRENTDIR = $(shell pwd)
SOURCEDIR = $(CURRENTDIR)
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')
APP_SOURCES := $(shell find $(SOURCEDIR) -name '*.go' -not -path '$(SOURCEDIR)/vendor/*')

PATH := $(CURRENTDIR)/bin:$(PATH)

COVERAGEDIR = $(CURRENTDIR)/coverage

VERSION = $(shell cat .goxc.json | python -c "import json,sys;obj=json.load(sys.stdin);print obj['PackageVersion'];")

TEMPDIR := $(shell mktemp -d)
LD_FLAGS = -ldflags '-w' -ldflags "-X main.VERSION=$(VERSION)"

TEST_TARGETS = $(PACKAGES)

all: build

deps:
	@./install_consul.sh
	@mkdir -p $(COVERAGEDIR)
	@which gover > /dev/null || \
        (go get github.com/modocache/gover)
	@which goxc > /dev/null || \
        (go get github.com/laher/goxc)

build-deps: deps format test check
	@mkdir -p bin/

build: build-deps
	go build $(LD_FLAGS) -o bin/marathon-consul

build-linux: build-deps
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo $(LD_FLAGS) -o bin/marathon-consul

docker: build-linux
	docker build -t allegro/marathon-consul .

test: deps $(SOURCES) $(TEST_TARGETS)
	gover $(COVERAGEDIR) $(COVERAGEDIR)/gover.coverprofile

$(TEST_TARGETS):
	go test -coverprofile=coverage/$(shell basename $@).coverprofile $(TESTARGS) $@

check-deps: deps
	@which gometalinter > /dev/null || \
        (go get github.com/alecthomas/gometalinter && gometalinter --install)

check: check-deps $(SOURCES) test
	gometalinter . --deadline  720s --vendor -D dupl -D gotype -D errcheck -D gas -D golint -E gofmt

format:
	goimports -w -l $(APP_SOURCES)

FPM-exists:
	@fpm -v || \
	(echo >&2 "FPM must be installed on the system. See https://github.com/jordansissel/fpm"; false)

deb: FPM-exists build
	mkdir -p dist/$(VERSION)/
	cd dist/$(VERSION)/ && \
	fpm -s dir \
	    -t deb \
        -n marathon-consul \
        -v $(VERSION) \
        --url="https://github.com/allegro/marathon-consul" \
        --vendor=Allegro \
        --maintainer="Allegro Group <opensource@allegro.pl>" \
        --description "Marathon-consul service (performs Marathon Tasks registration as Consul Services for service discovery) Marathon-consul takes information provided by the Marathon event bus and forwards it to Consul agents. It also re-syncs all the information from Marathon to Consul on startup and repeats it with given interval." \
        --deb-priority optional \
        --workdir $(TEMPDIR) \
        --license "Apache License, version 2.0" \
        ../../bin/marathon-consul=/usr/bin/marathon-consul \
        ../../debian/marathon-consul.service=/etc/systemd/system/marathon-consul.service \
        ../../debian/marathon-consul.upstart=/etc/init/marathon-consul.conf \
        ../../debian/config.json=/etc/marathon-consul.d/config.json

release: deb deps
	goxc

version: deps
	goxc -wc -pv=$(v)
	git add .goxc.json
	git commit -m "Release $(v)"
	git tag $(v)

.PHONY: all bump build release deb
