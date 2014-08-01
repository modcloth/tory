PACKAGE := github.com/modcloth-labs/tory
SUBPACKAGES := $(PACKAGE)/tory

VERSION_VAR := $(PACKAGE)/tory.VersionString
VERSION_VALUE := $(shell git describe --always --dirty --tags)

REV_VAR := $(PACKAGE)/tory.RevisionString
REV_VALUE := $(shell git rev-parse --sq HEAD)

BRANCH_VAR := $(PACKAGE)/tory.BranchString
BRANCH_VALUE := $(shell git rev-parse --abbrev-ref HEAD)

GENERATED_VAR := $(PACKAGE)/tory.GeneratedString
GENERATED_VALUE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

GO ?= go
GODEP ?= godep
GINKGO ?= ginkgo
GOBUILD_LDFLAGS := -ldflags "\
  -X $(VERSION_VAR) $(VERSION_VALUE) \
  -X $(REV_VAR) $(REV_VALUE) \
  -X $(BRANCH_VAR) $(BRANCH_VALUE) \
  -X $(GENERATED_VAR) $(GENERATED_VALUE)"
GOBUILD_FLAGS ?=
GINKGO_FLAGS ?=

.PHONY: all
all: clean build test save

.PHONY: build
build: deps
	$(GO) install $(GOBUILD_LDFLAGS) $(PACKAGE) $(SUBPACKAGES)

.PHONY: deps
deps:
	$(GODEP) restore
	$(GO) get github.com/modcloth-labs/json-server

.PHONY: test
test: build test-deps .test

.PHONY: .test
.test:
	$(GINKGO) $(GINKGO_FLAGS) -r --randomizeAllSpecs --failOnPending --cover --race

.PHONY: test-deps
test-deps:
	$(GO) test -i $(GOBUILD_LDFLAGS) $(PACKAGE) $(SUBPACKAGES)

.PHONY: clean
clean:
	$(RM) $${GOPATH%%:*}/bin/tory
	$(GO) clean -x $(PACKAGE) $(SUBPACKAGES)

.PHONY: save
save:
	$(GODEP) save -copy=false $(PACKAGE) $(SUBPACKAGES)
