PACKAGES=$(shell go list ./... | grep -v /vendor/)
TESTARGS?=-race
CURRENT_DIR = $(shell pwd)
SOURCEDIR = $(CURRENT_DIR)
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')
VERSION=$(shell cat .goxc.json | python -c "import json,sys;obj=json.load(sys.stdin);print obj['PackageVersion'];")
TEMPDIR := $(shell mktemp -d)
LD_FLAGS=-ldflags '-w' -ldflags "-X main.VERSION=$(VERSION)"

all: build

deps:
	@./install_consul.sh

build: deps test
	@mkdir -p bin/
	go build $(LD_FLAGS) -o bin/marathon-consul

build-linux: deps test
	@mkdir -p bin/
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo $(LD_FLAGS) -o bin/marathon-consul

docker: build-linux
	docker build -t allegro/marathon-consul .

test: deps $(SOURCES)
	PATH=$(CURRENT_DIR)/bin:$(PATH) go test $(PACKAGES) $(TESTARGS)
	go vet $(PACKAGES)

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

release: deb
	@go get github.com/laher/goxc
	goxc

bump:
	goxc bump
	git add .goxc.json
	git commit -m "Bumped version"

.PHONY: all build test xcompile package dist
