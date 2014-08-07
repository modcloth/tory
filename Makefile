PACKAGE := github.com/modcloth/tory
SUBPACKAGES := \
  $(PACKAGE)/tory \
  $(PACKAGE)/tory-ansible-inventory \
  $(PACKAGE)/tory-sync-from-joyent
ANSIBLE_MODULES := \
  $(PACKAGE)/ansible-modules/tory_add_host

COVERPROFILES := \
  main.coverprofile \
  tory.coverprofile \
  tory-ansible-inventory.coverprofile \
  tory-sync-from-joyent.coverprofile

VERSION_VAR := $(PACKAGE)/tory.VersionString
VERSION_VALUE := $(shell git describe --always --dirty --tags)

REV_VAR := $(PACKAGE)/tory.RevisionString
REV_VALUE := $(shell git rev-parse --sq HEAD)

BRANCH_VAR := $(PACKAGE)/tory.BranchString
BRANCH_VALUE := $(shell git rev-parse --abbrev-ref HEAD)

GENERATED_VAR := $(PACKAGE)/tory.GeneratedString
GENERATED_VALUE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

DOCKER_TAG ?= modcloth/tory:latest

DATABASE_URL ?= postgres://localhost/tory?sslmode=disable
PORT ?= 9462

DOCKER ?= docker
GO ?= go
GODEP ?= godep
GOBUILD_LDFLAGS := -ldflags "\
  -X $(VERSION_VAR) $(VERSION_VALUE) \
  -X $(REV_VAR) $(REV_VALUE) \
  -X $(BRANCH_VAR) $(BRANCH_VALUE) \
  -X $(GENERATED_VAR) $(GENERATED_VALUE)"
GOBUILD_FLAGS ?=
GOTEST_FLAGS ?= -race -v

QUIET ?=
VERBOSE ?=

export QUIET
export VERBOSE

.PHONY: all
all: clean build migrate test save

.PHONY: build
build: deps .build

.PHONY: .build
.build:
	$(GO) install $(GOBUILD_LDFLAGS) $(PACKAGE) $(SUBPACKAGES) $(ANSIBLE_MODULES)

.PHONY: deps
deps:
	$(GODEP) restore

.PHONY: test
test: build test-deps .test

.PHONY: .test
.test: coverage.html

coverage.html: all.coverprofile
	$(GO) tool cover -func=$<
	$(GO) tool cover -html=$< -o $@

all.coverprofile: $(COVERPROFILES)
	echo 'mode: count' > $@
	grep -h -v 'mode: count' $^ >> $@

main.coverprofile:
	$(GO) test $(GOTEST_FLAGS) $(GOBUILD_LDFLAGS) \
	  -coverprofile=$@ -covermode=count $(PACKAGE)

tory.coverprofile:
	$(GO) test $(GOTEST_FLAGS) $(GOBUILD_LDFLAGS) \
	  -coverprofile=$@ -covermode=count github.com/modcloth/tory/tory

tory-ansible-inventory.coverprofile:
	$(GO) test $(GOTEST_FLAGS) $(GOBUILD_LDFLAGS) \
	  -coverprofile=$@ -covermode=count github.com/modcloth/tory/tory-ansible-inventory

tory-sync-from-joyent.coverprofile:
	$(GO) test $(GOTEST_FLAGS) $(GOBUILD_LDFLAGS) \
	  -coverprofile=$@ -covermode=count github.com/modcloth/tory/tory-sync-from-joyent

.PHONY: migrate
migrate: build
	$${GOPATH%%:*}/bin/tory migrate -d $(DATABASE_URL)

.PHONY: test-deps
test-deps:
	$(GO) test -i $(GOTEST_FLAGS) $(GOBUILD_LDFLAGS) $(PACKAGE) $(SUBPACKAGES) $(ANSIBLE_MODULES)

.PHONY: clean
clean:
	$(RM) $${GOPATH%%:*}/bin/tory *.coverprofile coverage.html
	$(GO) clean -x $(PACKAGE) $(SUBPACKAGES) $(ANSIBLE_MODULES)

.PHONY: save
save:
	$(GODEP) save -copy=false $(PACKAGE) $(SUBPACKAGES) $(ANSIBLE_MODULES)

.PHONY: build-container
build-container:
	$(DOCKER) build -t $(DOCKER_TAG) .

.PHONY: run-container
run-container:
	$(DOCKER) run -p $(PORT):$(PORT) -e DATABASE_URL=$(DATABASE_URL) $(DOCKER_TAG)
