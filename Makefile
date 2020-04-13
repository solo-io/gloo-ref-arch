.PHONY: run-all
run-all:
	go test ./...

#----------------------------------------------------------------------------------
# Base
#----------------------------------------------------------------------------------

ROOTDIR := $(shell pwd)
OUTPUT_DIR:= $(ROOTDIR)/_output
PACKAGE_PATH:=github.com/solo-io/gloo-ref-arch
SOURCES := $(shell find . -name "*.go" | grep -v test.go | grep -v '\.\#*')
RELEASE := "true"
ifeq ($(TAGGED_VERSION),)
	TAGGED_VERSION := $(shell git describe --tags --dirty)
	RELEASE := "false"
endif
VERSION ?= $(shell echo $(TAGGED_VERSION) | cut -c 2-)

#----------------------------------------------------------------------------------
# Repo setup
#----------------------------------------------------------------------------------

# https://www.viget.com/articles/two-ways-to-share-git-hooks-with-your-team/
.PHONY: init
init:
	git config core.hooksPath .githooks

#------------
# Test server
#------------

.PHONY: build-test-server
build-test-server:
	GO111MODULE=on CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o _output/valet-test-server -v mtls/test-server/main.go

docker-build-test-server: build-test-server
	docker build -t quay.io/solo-io/valet-test-server:$(VERSION) -f mtls/test-server/Dockerfile _output

docker-push-test-server: docker-build-test-server
	docker push quay.io/solo-io/valet-test-server:$(VERSION)