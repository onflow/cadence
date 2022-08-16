#
# Cadence - The resource-oriented smart contract programming language
#
# Copyright 2019-2020 Dapper Labs, Inc.
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

COVERPKGS := $(shell go list ./... | grep -v /cmd | grep -v /runtime/test | tr "\n" "," | sed 's/,*$$//')


LINTERS :=
ifneq ($(linters),)
	LINTERS = -E $(linters)
endif


.PHONY: test
test:
	# test all packages
	GO111MODULE=on go test -coverprofile=coverage.txt -covermode=atomic -parallel 8 -race -coverpkg $(COVERPKGS) ./...
	# remove coverage of empty functions from report
	sed -i -e 's/^.* 0 0$$//' coverage.txt
	cd ./languageserver && make test

.PHONY: fast-test
fast-test:
	# test all packages
	GO111MODULE=on go test -parallel 8 ./...

.PHONY: build
build:
	go build -o ./runtime/cmd/parse/parse ./runtime/cmd/parse
	GOARCH=wasm GOOS=js go build -o ./runtime/cmd/parse/parse.wasm ./runtime/cmd/parse
	go build -o ./runtime/cmd/check/check ./runtime/cmd/check
	go build -o ./runtime/cmd/main/main ./runtime/cmd/main
	cd ./languageserver && make build

.PHONY: lint-github-actions
lint-github-actions: build-linter
	tools/golangci-lint/golangci-lint run --out-format=github-actions --timeout=5m  -v ./...

.PHONY: lint
lint: build-linter
	tools/golangci-lint/golangci-lint run $(LINTERS) --timeout=5m -v ./...


.PHONY: fix-lint
fix-lint: build-linter
	tools/golangci-lint/golangci-lint run -v --fix --timeout=5m  $(LINTERS) ./...

.PHONY: build-linter
build-linter: tools/golangci-lint/golangci-lint tools/maprangecheck/maprangecheck.so tools/constructorcheck/constructorcheck.so

tools/maprangecheck/maprangecheck.so:
	(cd tools/maprangecheck && $(MAKE) plugin)

tools/constructorcheck/constructorcheck.so:
	(cd tools/constructorcheck && $(MAKE) plugin)

tools/golangci-lint/golangci-lint:
	(cd tools/golangci-lint && $(MAKE))

.PHONY: check-headers
check-headers:
	@./check-headers.sh

.PHONY: generate
generate:
	go generate -v ./...

.PHONY: check-tidy
check-tidy: generate
	go mod tidy
	cd languageserver; go mod tidy
	git diff --exit-code

.PHONY: release
release:
	@(VERSIONED_FILES="version.go \
	npm-packages/cadence-parser/package.json \
	npm-packages/cadence-docgen/package.json" \
	./bump-version.sh $(bump))

.PHONY: check-capabilities
check-capabilities:
	go install github.com/cugu/gocap@v0.1.0
	go mod download
	gocap check .
