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

COVERPKGS := $(shell go list ./... | grep -v /cmd | grep -v /test | tr "\n" "," | sed 's/,*$$//')


LINTERS :=
ifneq ($(linters),)
	LINTERS = -E $(linters)
endif

.PHONY: build
build: build-tools ./cmd/parse/parse ./cmd/parse/parse.wasm ./cmd/check/check ./cmd/main/main

./cmd/parse/parse:
	go build -o $@ ./cmd/parse

./cmd/parse/parse.wasm:
	GOARCH=wasm GOOS=js go build -o $@ ./cmd/parse

./cmd/check/check:
	go build -o $@ ./cmd/check

./cmd/main/main:
	go build -o $@ ./cmd/main

.PHONY: build-tools
build-tools: build-analysis build-get-contracts build-compatibility-check

.PHONY: build-analysis
build-analysis:
	(cd ./tools/analysis && go build .)

.PHONY: build-get-contracts
build-get-contracts:
	(cd ./tools/get-contracts && go build .)

.PHONY: build-compatibility-check
build-compatibility-check:
	(cd ./tools/compatibility-check && go build .)

.PHONY: ci
ci:
	# test all packages
	go test -coverprofile=coverage.txt -covermode=atomic -parallel 8 -race -coverpkg $(COVERPKGS) ./...
	# run interpreter smoke tests. results from run above are reused, so no tests runs are duplicated
	go test -count=5 ./interpreter/... -runSmokeTests=true -validateAtree=false
	# remove coverage of empty functions from report
	sed -i -e 's/^.* 0 0$$//' coverage.txt

.PHONY: test
test: test-all-packages test-tools

.PHONY: test-all-packages
test-all-packages:
	(go test -parallel 8 ./...)

.PHONY: test-tools
test-tools:
	(cd ./tools/analysis && go test -parallel 8 ./)
	(cd ./tools/compatibility-check && go test -parallel 8 ./)
	(cd ./tools/constructorcheck && go test -parallel 8 ./)
	(cd ./tools/maprange && go test -parallel 8 ./)

.PHONY: lint-github-actions
lint-github-actions: build-linter
	tools/golangci-lint/golangci-lint run --out-format=colored-line-number,github-actions --timeout=5m  -v ./...

.PHONY: lint
lint: build-linter
	tools/golangci-lint/golangci-lint run $(LINTERS) --timeout=5m -v ./...

.PHONY: fix-lint
fix-lint: build-linter
	tools/golangci-lint/golangci-lint run -v --fix --timeout=5m  $(LINTERS) ./...

.PHONY: build-linter
build-linter: tools/golangci-lint/golangci-lint tools/maprange/maprange.so tools/unkeyed/unkeyed.so tools/constructorcheck/constructorcheck.so

tools/maprange/maprange.so:
	(cd tools/maprange && $(MAKE))

tools/unkeyed/unkeyed.so:
	(cd tools/unkeyed && $(MAKE))

tools/constructorcheck/constructorcheck.so:
	(cd tools/constructorcheck && $(MAKE))

tools/golangci-lint/golangci-lint:
	(cd tools/golangci-lint && $(MAKE))

.PHONY: clean-linter
clean-linter:
	rm -f tools/golangci-lint/golangci-lint \
		tools/maprange/maprange.so \
		tools/unkeyed/unkeyed.so \
		tools/constructorcheck/constructorcheck.so

.PHONY: check-headers
check-headers:
	@./check-headers.sh

.PHONY: generate
generate:
	go generate -v ./...

.PHONY: check-tidy
check-tidy: generate
	go mod tidy
	git diff --exit-code

.PHONY: release
release:
	@(VERSIONED_FILES="version.go \
	npm-packages/cadence-parser/package.json" \
	bash ./bump-version.sh $(bump))
