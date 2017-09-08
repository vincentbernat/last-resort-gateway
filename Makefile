DATE ?= $(shell date +%FT%T%z)
VERSION ?= $(shell git describe --tags --always --dirty 2> /dev/null || cat $(CURDIR)/.version 2> /dev/null || echo v0)
GOPATH = $(CURDIR)/.gopath~
BIN = $(GOPATH)/bin
PACKAGE = lrg
BASE = $(GOPATH)/src/$(PACKAGE)
PKGS = $(or $(PKG),$(shell cd $(BASE) && env GOPATH=$(GOPATH) $(GO) list ./... | grep -v /vendor/))
TESTPKGS = $(shell $(GO) list -f '{{ if .TestGoFiles }}{{ .ImportPath }}{{end}}' $(PKGS))

export VERSION
export GOPATH

GO = go
GOFMT = gofmt
TIMEOUT = 30
M = $(shell printf "\033[34;1m▶\033[0m")

.PHONY: all
all: fmt test last-resort-gateway docs ; $(info $(M) All done!) @ ## Run all tests, build lrg and documentation

.PHONY: last-resort-gateway
last-resort-gateway: vendor | $(BASE) ; $(info $(M) building executable…) @ ## Build last-resort-gateway binary
	cd $(BASE) && $(GO) build \
		-tags release \
		-ldflags '-X $(PACKAGE)/cmd.Version=$(VERSION) -X $(PACKAGE)/cmd.BuildDate=$(DATE)' \
		-o bin/$@ main.go

# Tools

GODEP = $(BIN)/dep
$(BIN)/dep: | $(BASE) ; $(info $(M) building go dep…)
	@go get github.com/golang/dep/cmd/dep

GOLINT = $(BIN)/golint
$(BIN)/golint: | $(BASE) ; $(info $(M) building golint…)
	@go get github.com/golang/lint/golint

GOCOVMERGE = $(BIN)/gocovmerge
$(BIN)/gocovmerge: | $(BASE) ; $(info $(M) building gocovmerge…)
	@go get github.com/wadey/gocovmerge

GOCOV = $(BIN)/gocov
$(BIN)/gocov: | $(BASE) ; $(info $(M) building gocov…)
	@go get github.com/axw/gocov/...

GOCOVXML = $(BIN)/gocov-xml
$(BIN)/gocov-xml: | $(BASE) ; $(info $(M) building gocov-xml…)
	@go get github.com/AlekSi/gocov-xml

# Namespacing stuff. Create a throwaway namespace useful for tests.

.PHONY: namespace namespace-setup
namespace: ; $(info $(M) switching to isolated namespace…)
	@set -e ; for args in "--net --mount --ipc" "--net --mount --ipc --user --map-root-user" ""; do \
		! unshare $$args true 2> /dev/null || break ; \
	 done ; \
	 if test x"$$args" = x; then \
		2>&1 printf "\033[31;1m⚠\033[0m Throwaway namespace not available, check README\n"; \
		false ; \
	 fi ; \
	 unshare $$args -- $(MAKE) --no-print-directory namespace-setup $(TARGET) \
		NAME=$(NAME) ARGS="$(ARGS)"
namespace-setup: ; $(info $(M) setting up isolated namespace…)
	@ip link set up dev lo
	@mount -n --make-rprivate /
	@mount -n -t sysfs sysfs /sys
	@mount -n -t tmpfs tmpfs /tmp
	@mount -n -t tmpfs tmpfs /var/run
	@mount -n -t tmpfs tmpfs /root
	@mount -n --bind config/testdata/rt_protos /etc/iproute2/rt_protos
	@mount -n --bind config/testdata/rt_tables /etc/iproute2/rt_tables

# The tests-* targets just set the ARGS variable and depends on
# test. The test target will setup an isolated network namespace
# before running tests.

TEST_TARGETS := test-default test-bench test-short test-verbose test-race
.PHONY: $(TEST_TARGETS) test test--
test tests: TARGET=test--
test tests: fmt lint vendor | $(BASE) namespace	## Run tests
test-bench: ARGS=-run=__absolutelynothing__ -bench=.    ## Run benchmarks
test-short: ARGS=-short               ## Run only short tests
test-verbose: ARGS=-v                 ## Run tests in verbose mode with coverage reporting
test-race: ARGS=-race                 ## Run tests with race detector
$(TEST_TARGETS): NAME=$(MAKECMDGOALS:test-%=%)
$(TEST_TARGETS): test$(NAMESPACE:%=--%)
test--: ; $(info $(M) running $(NAME:%=% )tests…)
	@cd $(BASE) && $(GO) test -timeout $(TIMEOUT)s $(ARGS) $(TESTPKGS)

COVERAGE_MODE = atomic
COVERAGE_PROFILE = $(COVERAGE_DIR)/profile.out
COVERAGE_XML = $(COVERAGE_DIR)/coverage.xml
COVERAGE_HTML = $(COVERAGE_DIR)/index.html
.PHONY: test-coverage test-coverage--
test-coverage: TARGET=test-coverage--
test-coverage: fmt lint vendor $(GOCOVMERGE) $(GOCOV) $(GOCOVXML) | $(BASE) namespace ## Run coverage tests
test-coverage--: COVERAGE_DIR:=$(shell readlink -f test/coverage.$$(date -Iseconds))
test-coverage--: ; $(info $(M) running coverage tests…)
	@mkdir -p $(COVERAGE_DIR)/coverage
	@cd $(BASE) && for pkg in $(TESTPKGS); do \
		$(GO) test \
			-coverpkg=$$($(GO) list -f '{{ join .Deps "\n" }}' $$pkg | \
					grep '^$(PACKAGE)/' | grep -v '^$(PACKAGE)/vendor/' | \
					tr '\n' ',')$$pkg \
			-covermode=$(COVERAGE_MODE) \
			-coverprofile="$(COVERAGE_DIR)/coverage/`echo $$pkg | tr "/" "-"`.cover" $$pkg ;\
	 done
	@$(GOCOVMERGE) $(COVERAGE_DIR)/coverage/*.cover > $(COVERAGE_PROFILE)
	@$(GO) tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	@$(GOCOV) convert $(COVERAGE_PROFILE) | $(GOCOVXML) > $(COVERAGE_XML)

.PHONY: lint
lint: vendor $(GOLINT) | $(BASE) ; $(info $(M) running golint…) @ ## Run golint
	@cd $(BASE) && ret=0 && for pkg in $(PKGS); do \
		test -z "$$($(GOLINT) $$pkg | tee /dev/stderr)" || ret=1 ; \
	 done ; exit $$ret

.PHONY: vet
vet: vendor | $(BASE) ;  $(info $(M) running go vet…) @ ## Run go vet
	@cd $(BASE) && $(GO) vet $(PKGS)

.PHONY: fmt
fmt: ; $(info $(M) running gofmt…) @ ## Run gofmt on all source files
	@ret=0 && for d in $$($(GO) list -f '{{.Dir}}' ./... | grep -v /vendor/); do \
		$(GOFMT) -l -w $$d/*.go || ret=$$? ; \
	 done ; exit $$ret

vendor: Gopkg.toml Gopkg.lock | $(BASE) $(GODEP) ; $(info $(M) retrieving dependencies…)
	@cd $(BASE) && $(GODEP) ensure
	@ln -nsf . vendor/src
	@touch $@
.PHONY: vendor-update
vendor-update: | $(BASE) $(GODEP)
ifeq "$(origin PKG)" "command line"
	$(info $(M) updating $(PKG) dependency…)
	@cd $(BASE) && $(GODEP) ensure -update $(PKG)
else
	$(info $(M) updating all dependencies…)
	@cd $(BASE) && $(GODEP) ensure -update
endif
	@ln -nsf . vendor/src
	@touch vendor

.PHONY: docs doc
docs doc: ; $(info $(M) building documentation…) @ ## Build documentation
	$(MAKE) -C docs html

.PHONY: clean
clean: ; $(info $(M) cleaning…)	@ ## Cleanup everything
	rm -rf $(GOPATH)
	rm -rf bin
	rm -rf test/test.* test/coverage.*
	$(MAKE) -C docs clean

$(BASE): ; $(info $(M) setting GOPATH…) @ ## Setup GOPATH
	@mkdir -p $(dir $@)
	@ln -sf $(CURDIR) $@

.PHONY: help
help:
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: version
version:
	@echo $(VERSION)

.PHONY: metrics
metrics:
	@cd $(BASE) && for p in $(PKGS); do for f in $$($(GO) list -f '{{join .GoFiles " "}}' $$p); do \
		p=$${p#$(PACKAGE)}; p=$${p#/}; f=$${f%.go}; \
		sed -n 's+.*r\.\(Counter\|Gauge\|GaugeFloat64\|Histogram\|Meter\|Timer\)(.*\?"\(.*\)".*+\1\t'$$p.'\2+p' ./$$p/$$f.go | sed 's+/+.+g' | sort | uniq; \
	done; done
