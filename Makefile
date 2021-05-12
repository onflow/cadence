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

.PHONY: test
test:
	# test all packages
	GO111MODULE=on go test $(if $(JSON_OUTPUT),-json,) -parallel 8 -race ./...
	cd ./languageserver && make test

.PHONY: build
build:
	go build -o ./runtime/cmd/parse/parse ./runtime/cmd/parse
	GOARCH=wasm GOOS=js go build -o ./runtime/cmd/parse/parse.wasm ./runtime/cmd/parse
	go build -o ./runtime/cmd/check/check ./runtime/cmd/check
	go build -o ./runtime/cmd/main/main ./runtime/cmd/main
	cd ./languageserver && make build

.PHONY: lint-github-actions
lint-github-actions: build-linter
	tools/golangci-lint/golangci-lint run --out-format=github-actions -v ./...

.PHONY: lint
lint: build-linter
	tools/golangci-lint/golangci-lint run -v ./...

.PHONY: build-linter
build-linter: tools/golangci-lint/golangci-lint tools/maprangecheck/maprangecheck.so

tools/maprangecheck/maprangecheck.so:
	(cd tools/maprangecheck && $(MAKE) plugin)

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
	npm-packages/cadence-language-server/package.json" \
	./bump-version.sh $(bump))
