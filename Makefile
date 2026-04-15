#
# Cadence - The resource-oriented smart contract programming language
#
# Copyright Flow Foundation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

GOPATH ?= $(HOME)/go

# Ensure go bin path is in path (Especially for CI)
PATH := $(PATH):$(GOPATH)/bin

LINTERS :=
ifneq ($(linters),)
	LINTERS = -E $(linters)
endif

.PHONY: build
build: build-commands build-tools

# Commands

.PHONY: build-commands
build-commands: ./cmd/parse/parse ./cmd/parse/parse.wasm ./cmd/check/check ./cmd/main/main

./cmd/parse/parse:
	go build -o $@ ./cmd/parse

./cmd/parse/parse.wasm:
	GOARCH=wasm GOOS=js go build -o $@ ./cmd/parse

./cmd/check/check:
	go build -o $@ ./cmd/check

./cmd/main/main:
	go build -o $@ ./cmd/main

# Tools

.PHONY: build-tools
build-tools: build-analysis build-get-contracts build-compatibility-check

.PHONY: test-tools
test-tools: test-analysis test-compatibility-check test-subtype-gen

## Analysis tool

.PHONY: build-analysis
build-analysis:
	(cd ./tools/analysis && go build .)

.PHONY: test-analysis
test-analysis:
	(cd ./tools/analysis && go test .)

## Get contracts tool

.PHONY: build-get-contracts
build-get-contracts:
	(cd ./tools/get-contracts && go build .)

## Compatibility check tool

.PHONY: build-compatibility-check
build-compatibility-check:
	(cd ./tools/compatibility-check && go build .)

.PHONY: test-compatibility-check
test-compatibility-check:
	(cd ./tools/compatibility-check && go test .)

## Subtyping generator tool

.PHONY: test-subtype-gen
test-subtype-gen:
	(cd ./tools/subtype-gen && go test .)

# Testing

TEST_PKGS := $(shell go list ./... | grep -Ev '/cmd|/analysis|/tools')
COVER_PKGS := $(shell echo $(TEST_PKGS) | tr ' ' ',')

.PHONY: test
test: test-with-compiler test-with-tracing
	go test -tags compare_subtyping $(TEST_PKGS)

.PHONY: test-with-tracing
test-with-tracing:
	go test -tags cadence_tracing $(TEST_PKGS)

.PHONY: ci
ci: test-with-coverage test-with-compiler smoke-test

.PHONY: ci-with-tracing
ci-with-tracing: test-with-tracing test-with-compiler-and-tracing

.PHONY: test-with-coverage
test-with-coverage:
	go test -tags compare_subtyping -coverprofile=coverage.txt -covermode=atomic -race -coverpkg $(COVER_PKGS) $(TEST_PKGS)
	# remove coverage of empty functions from report
	sed -i -e 's/^.* 0 0$$//' coverage.txt

.PHONY: smoke-test
smoke-test:
	go test -count=5 ./interpreter/... -runSmokeTests=true -validateAtree=false

.PHONY: test-with-compiler
test-with-compiler:
	go test ./interpreter/... ./runtime/... -compile=true

.PHONY: test-with-compiler-and-tracing
test-with-compiler-and-tracing:
	go test -tags cadence_tracing ./interpreter/... ./runtime/... -compile=true

# Benchmarking

BENCH_REPS ?= 1
BENCH_TIME ?= 3s
BENCH_PKGS ?= $(shell go list ./... | grep -Ev '/old_parser')
BENCH_PKGS_COMMON ?= $(shell go list ./... | grep -Ev '/old_parser|/encoding|/parser|/sema|/bbq')

.PHONY: bench
bench:
	for i in {1..$(BENCH_REPS)}; do \
		go test -run=^$$ -bench=. -benchmem -shuffle=on -benchtime=$(BENCH_TIME) $(BENCH_PKGS) ; \
	done

.PHONY: bench-common
bench-common:
	$(MAKE) bench BENCH_PKGS="$(BENCH_PKGS_COMMON)"

# Linting

.PHONY: lint
lint: build-linter
	tools/golangci-lint/golangci-lint run $(LINTERS) --timeout=5m -v ./...

.PHONY: fix-lint
fix-lint: build-linter
	tools/golangci-lint/golangci-lint run -v --fix --timeout=5m  $(LINTERS) ./...

.PHONY: build-linter
build-linter: tools/golangci-lint/golangci-lint tools/maprange/maprange.so tools/unkeyed/unkeyed.so tools/constructorcheck/constructorcheck.so

.PHONY: test-linter
test-linter: test-maprange test-unkeyed test-constructorcheck

.PHONY: clean-linter
clean-linter: clean-maprange clean-unkeyed clean-constructorcheck
	rm -f tools/golangci-lint/golangci-lint

## Maprange linter

tools/maprange/maprange.so:
	(cd tools/maprange && $(MAKE))

.PHONY: clean-maprange
clean-maprange:
	rm -f tools/maprange/maprange.so

.PHONY: test-maprange
test-maprange:
	(cd ./tools/maprange && go test .)

## Unkeyed linter

tools/unkeyed/unkeyed.so:
	(cd tools/unkeyed && $(MAKE))

.PHONY: clean-unkeyed
clean-unkeyed:
	rm -f tools/unkeyed/unkeyed.so

.PHONY: test-unkeyed
test-unkeyed:
	(cd ./tools/unkeyed && go test .)

## Constructorcheck linter

tools/constructorcheck/constructorcheck.so:
	(cd tools/constructorcheck && $(MAKE))

.PHONY: test-constructorcheck
test-constructorcheck:
	(cd ./tools/constructorcheck && go test .)

tools/golangci-lint/golangci-lint:
	(cd tools/golangci-lint && $(MAKE))

.PHONY: clean-constructorcheck
clean-constructorcheck:
	rm -f tools/constructorcheck/constructorcheck.so

# Code generation

.PHONY: generate
generate:
	go install golang.org/x/tools/cmd/stringer@v0.32.0
	go generate -v ./...

# Other checks and validation

.PHONY: check-headers
check-headers:
	@./check-headers.sh

.PHONY: check-tidy
check-tidy: generate
	go mod tidy
	git diff --exit-code

.PHONY: validate-error-doc-links
validate-error-doc-links:
	go run ./cmd/errors validate-doc-links

# Release (version bumping)

.PHONY: release
release:
	@(VERSIONED_FILES="version.go \
	npm-packages/cadence-parser/package.json" \
	bash ./bump-version.sh $(bump))

# Tools

.PHONY: install-benchstat
install-benchstat:
	# Last version to support HTML output
	go install golang.org/x/perf/cmd/benchstat@91a04616dc65ba76dbe9e5cf746b923b1402d303
