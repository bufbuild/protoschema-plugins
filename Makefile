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
COPYRIGHT_YEARS := 2024
LICENSE_IGNORE := --ignore testdata/

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
	go run internal/cmd/pubsub-generate-testdata/main.go internal/testdata/pubsub
	go run internal/cmd/jsonschema-generate-testdata/main.go internal/testdata/jsonschema

.PHONY: build
build: generate ## Build all packages
	go build ./...

.PHONY: lint
lint: $(BIN)/golangci-lint $(BIN)/buf ## Lint
	go vet ./...
	$(BIN)/golangci-lint run
	buf lint
	buf format -d --exit-code

.PHONY: lintfix
lintfix: $(BIN)/golangci-lint ## Automatically fix some lint errors
	$(BIN)/golangci-lint run --fix
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
	go get -u -t ./...
	go mod tidy -v
	buf mod update internal/proto

.PHONY: checkgenerate
checkgenerate:
	@# Used in CI to verify that `make generate` doesn't produce a diff.
	test -z "$$(git status --porcelain | tee /dev/stderr)"

$(BIN):
	@mkdir -p $(BIN)

$(BIN)/buf: $(BIN) Makefile
	go install github.com/bufbuild/buf/cmd/buf@latest

$(BIN)/license-header: $(BIN) Makefile
	go install github.com/bufbuild/buf/private/pkg/licenseheader/cmd/license-header@latest

$(BIN)/golangci-lint: $(BIN) Makefile
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.57.1

$(BIN)/jv: $(BIN) Makefile
	go install github.com/santhosh-tekuri/jsonschema/cmd/jv@latest
