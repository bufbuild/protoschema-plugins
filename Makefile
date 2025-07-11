# See https://tech.davis-hansson.com/p/make/
SHELL := bash
.DELETE_ON_ERROR:
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := all
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules
MAKEFLAGS += --no-print-directory
BIN := .tmp/bin
export PATH := $(BIN):$(PATH)
export GOBIN := $(abspath $(BIN))
BUF_VERSION = $(shell go list -m -f '{{.Version}}' github.com/bufbuild/buf)
COPYRIGHT_YEARS := 2024-2025
GOLANGCI_LINT_VERSION := v2.1.6
GOLANGCI_LINT := $(BIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)
LICENSE_IGNORE := --ignore testdata/

UNAME_OS := $(shell uname -s)
ifeq ($(UNAME_OS),Darwin)
# Explicitly use the "BSD" sed shipped with Darwin. Otherwise if the user has a
# different sed (such as gnu-sed) on their PATH this will fail in an opaque
# manner. /usr/bin/sed can only be modified if SIP is disabled, so this should
# be relatively safe.
SED_I := /usr/bin/sed -i ''
else
SED_I := sed -i
endif

.PHONY: help
help: ## Describe useful make targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "%-30s %s\n", $$1, $$2}'

.PHONY: all
all: ## Build, test, and lint (default)
	$(MAKE) test
	$(MAKE) lint

.PHONY: clean
clean: ## Delete intermediate build artifacts
	@# -X only removes untracked files, -d recurses into directories, -f actually removes files/dirs
	git clean -Xdf

.PHONY: test
test: build $(BIN)/jv ## Run unit tests
	go test -vet=off -race -cover ./...
	$(BIN)/jv internal/testdata/jsonschema/bufext.cel.expr.conformance.proto3.TestAllTypes.schema.json internal/testdata/jsonschema-doc/test.TestAllTypes.yaml

.PHONY: golden
golden: generate
	rm -rf internal/testdata/pubsub
	rm -rf internal/testdata/jsonschema
	buf build ./internal/proto -o -#format=json > ./internal/testdata/codegenrequest/input.json
	buf generate
	go run internal/cmd/pubsub-generate-testdata/main.go internal/testdata/pubsub
	go run internal/cmd/jsonschema-generate-testdata/main.go internal/testdata/jsonschema

.PHONY: build
build: generate ## Build all packages
	go build ./...

.PHONY: lint
lint: $(GOLANGCI_LINT) $(BIN)/buf ## Lint
	go vet ./...
	$(GOLANGCI_LINT) fmt --diff
	$(GOLANGCI_LINT) run
	buf lint
	buf format -d --exit-code

.PHONY: lintfix
lintfix: $(GOLANGCI_LINT) ## Automatically fix some lint errors
	$(GOLANGCI_LINT) fmt
	$(GOLANGCI_LINT) run --fix
	buf format -w

.PHONY: install
install: ## Install all binaries
	go install ./...

.PHONY: generate
generate: $(BIN)/license-header $(BIN)/buf ## Regenerate code and licenses
	rm -rf internal/gen
	buf generate
	license-header \
		--license-type apache \
		--copyright-holder "Buf Technologies, Inc." \
		--year-range "$(COPYRIGHT_YEARS)" $(LICENSE_IGNORE)

.PHONY: upgrade
upgrade: ## Upgrade dependencies
	@UPDATE_PKGS=$$(go list -u -f '{{if and .Update (not (or .Main .Indirect .Replace))}}{{.Path}}@{{.Update.Version}}{{end}}' -m all); \
	if [[ -n "$${UPDATE_PKGS}" ]]; then \
		go get $${UPDATE_PKGS}; \
		go mod tidy -v; \
	fi
	buf dep update internal/proto
	# Update protobuf version to match version in go.mod after upgrade
	PROTOBUF_VERSION=$$(go list -m -f '{{.Version}}' google.golang.org/protobuf); \
	if [[ "$${PROTOBUF_VERSION}" =~ ^v[[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+$$ ]]; then \
		$(SED_I) -e "s|buf.build/protocolbuffers/go:.*|buf.build/protocolbuffers/go:$${PROTOBUF_VERSION}|" buf.gen.yaml; \
	fi

.PHONY: checkgenerate
checkgenerate:
	@# Used in CI to verify that `make generate` doesn't produce a diff.
	test -z "$$(git status --porcelain | tee /dev/stderr)"

$(BIN):
	@mkdir -p $(BIN)

$(BIN)/buf: $(BIN) Makefile
	go install github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)

$(BIN)/license-header: $(BIN) Makefile
	go install github.com/bufbuild/buf/private/pkg/licenseheader/cmd/license-header@$(BUF_VERSION)

$(GOLANGCI_LINT): $(BIN) Makefile
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	mv $(BIN)/golangci-lint $@

$(BIN)/jv: $(BIN) Makefile
	go install github.com/santhosh-tekuri/jsonschema/cmd/jv@latest
