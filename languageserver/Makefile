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


.PHONY: build
build:
	go build -o ./cmd/languageserver/languageserver ./cmd/languageserver
	GOARCH=wasm GOOS=js go build -o ./cmd/languageserver/languageserver.wasm ./cmd/languageserver

.PHONY: generate
generate:
	go get -d github.com/vektra/mockery/cmd/mockery
	go generate ./...

.PHONY: test
test:
	cd test && npm install && npm t

.PHONY: wasm
wasm:
	GOARCH=wasm GOOS=js go build -o languageserver.wasm  ./cmd/languageserver