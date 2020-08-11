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

# Disable go sum database lookup for private repos
GOPRIVATE=github.com/onflow/*

GOPATH ?= $(HOME)/go

# Ensure go bin path is in path (Especially for CI)
PATH := $(PATH):$(GOPATH)/bin

.PHONY: test
test:
	# test all packages
	GO111MODULE=on go test $(if $(JSON_OUTPUT),-json,) -parallel 8 ./...

.PHONY: build
build:
	go build -o ./runtime/cmd/parse/parse ./runtime/cmd/parse
	go build -o ./runtime/cmd/check/check ./runtime/cmd/check
	go build -o ./runtime/cmd/main/main ./runtime/cmd/main
	(cd languageserver && go build -o ./cmd/languageserver/languageserver ./cmd/languageserver)
	(cd languageserver && GOARCH=wasm GOOS=js go build -o ./cmd/languageserver/languageserver.wasm ./cmd/languageserver)

.PHONY: install-tools
install-tools:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${GOPATH}/bin v1.26.0

.PHONY: lint
lint:
	golangci-lint run -v ./...

.PHONY: check-headers
check-headers:
	@./check-headers.sh

.PHONY: check-tidy
check-tidy:
	go generate -v ./...
	go mod tidy
	cd languageserver; go mod tidy
	git diff --exit-code