PACKAGE := github.com/modcloth/tory
SUBPACKAGES := $(PACKAGE)/tory

COVERPROFILES := \
  main.coverprofile \
  tory.coverprofile

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
FLAKE8 ?= flake8
GO ?= go
GOX ?= gox
GODEP ?= godep
GO_BINDATA ?= go-bindata
PIP ?= pip
PYTEST ?= py.test
ifeq ($(shell uname),Darwin)
SHA256SUM ?= gsha256sum
else
SHA256SUM ?= sha256sum
endif
GOBUILD_LDFLAGS := -ldflags "\
  -X $(VERSION_VAR) $(VERSION_VALUE) \
  -X $(REV_VAR) $(REV_VALUE) \
  -X $(BRANCH_VAR) $(BRANCH_VALUE) \
  -X $(GENERATED_VAR) $(GENERATED_VALUE) \
  -w -s"
GOBUILD_FLAGS ?= -tags 'netgo'
GOTEST_FLAGS ?= -v

GOX_OSARCH ?= linux/amd64 darwin/amd64 windows/amd64
GOX_FLAGS ?= \
	-output="tory-{{.OS}}-{{.Arch}}/bin/{{.Dir}}" \
	-osarch="$(GOX_OSARCH)"

CROSS_TARBALLS := \
	tory-linux-amd64.tar.bz2 \
	tory-darwin-amd64.tar.bz2 \
	tory-windows-amd64.tar.bz2
PYTEST_FLAGS ?= \
	--cov-report term-missing \
	--cov tory_sync_from_joyent \
	--cov tory_register \
	--cov tory_inventory \
	--pep8 -rs --pdb
ALLFILES := $(shell git ls-files)
PYFILES := $(shell grep -l -E '^\#!/usr/bin/env python' $(ALLFILES))

CGO_ENABLED ?= 0
QUIET ?=
VERBOSE ?=

export CGO_ENABLED
export QUIET
export VERBOSE

.PHONY: all
all: clean build migrate test save pycheck pytest

.PHONY: build
build: deps .build

.PHONY: .build
.build:
	$(GO) install -a $(GOBUILD_FLAGS) $(GOBUILD_LDFLAGS) $(PACKAGE) $(SUBPACKAGES)

.PHONY: deps
deps: tory/bindata.go
	$(GODEP) restore

tory/bindata.go: .go-bindata-bootstrap $(wildcard public/*)
	$(GO_BINDATA) -prefix=public -o=$@ -pkg=tory ./public

.go-bindata-bootstrap:
	$(GO) get -x $(GOBUILD_FLAGS) github.com/jteeuwen/go-bindata/go-bindata > $@

.PHONY: crossbuild
crossbuild: deps .gox-bootstrap
	$(GOX) $(GOX_FLAGS) $(GOBUILD_LDFLAGS) $(PACKAGE) $(SUBPACKAGES)

.PHONY: crosstars
crosstars: $(CROSS_TARBALLS) SHA256SUMS

SHA256SUMS: $(CROSS_TARBALLS)
	$(SHA256SUM) $(CROSS_TARBALLS) > $@

tory-linux-amd64.tar.bz2: crossbuild
	rsync -av hosts bin tory-linux-amd64/
	tar -cjvf $@ tory-linux-amd64

tory-darwin-amd64.tar.bz2: crossbuild
	rsync -av hosts bin tory-darwin-amd64/
	tar -cjvf $@ tory-darwin-amd64

tory-windows-amd64.tar.bz2: crossbuild
	cp -v bin/tory-sync-from-joyent tory-windows-amd64/bin/tory-sync-from-joyent.py
	mkdir -p tory-windows-amd64/hosts
	cp -v hosts/tory tory-windows-amd64/hosts/tory.py
	tar -cjvf $@ tory-windows-amd64

.gox-bootstrap:
	$(GOX) -build-toolchain -osarch="$(GOX_OSARCH)" -verbose 2>&1 | tee $@

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

.PHONY: migrate
migrate: build
	$${GOPATH%%:*}/bin/tory migrate -d $(DATABASE_URL)

.PHONY: test-deps
test-deps:
	$(GO) test -i $(GOTEST_FLAGS) $(GOBUILD_LDFLAGS) $(PACKAGE) $(SUBPACKAGES)

.PHONY: clean
clean:
	$(RM) -r .coverage .*-bootstrap .cache/ $(shell find . -name '*.pyc')
	$(RM) $${GOPATH%%:*}/bin/tory *.coverprofile coverage.html
	$(RM) -r tory-*-amd64*
	$(GO) clean -x $(PACKAGE) $(SUBPACKAGES)

.PHONY: distclean
distclean: clean
	$(RM) .gox-bootstrap .go-bindata-bootstrap .flake8-bootstrap

.PHONY: save
save:
	$(GODEP) save -copy=false $(PACKAGE) $(SUBPACKAGES)

.PHONY: pycheck
pycheck: .flake8-bootstrap
	$(FLAKE8) $(PYFILES)

.flake8-bootstrap:
	(flake8 --version || $(PIP) install -r requirements.txt) && touch $@

.PHONY: pytest
pytest: .pytest-bootstrap
	$(PYTEST) $(PYTEST_FLAGS) tests/

.pytest-bootstrap:
	(py.test --version || $(PIP) install -r requirements.txt) && touch $@

.PHONY: build-container
build-container:
	$(DOCKER) build -t $(DOCKER_TAG) .

.PHONY: run-container
run-container:
	$(DOCKER) run -p $(PORT):$(PORT) -e DATABASE_URL=$(DATABASE_URL) $(DOCKER_TAG)
