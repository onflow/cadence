# Name of the cover profile
COVER_PROFILE := cover.out

# Disable go sum database lookup for private repos
GOPRIVATE=github.com/dapperlabs/*

GOPATH ?= $(HOME)/go

# Ensure go bin path is in path (Especially for CI)
PATH := $(PATH):$(GOPATH)/bin

.PHONY: test
test:
	# test all packages
	GO111MODULE=on go test -coverprofile=$(COVER_PROFILE) $(if $(JSON_OUTPUT),-json,) ./...

.PHONY: install-tools
install-tools:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${GOPATH}/bin v1.23.8

.PHONY: lint
lint:
	golangci-lint run -v ./...

# this ensures there is no unused dependency being added by accident
.PHONY: tidy
tidy:
	go mod tidy; git diff --exit-code

.PHONY: coverage
coverage:
ifeq ($(COVER), true)
	# Cover summary has to produce cover.json
	COVER_PROFILE=$(COVER_PROFILE) ./cover-summary.sh
	# file has to be called index.html
	gocov-html cover.json > index.html
	# coverage.zip will automatically be picked up by teamcity
	zip coverage.zip index.html
endif

.PHONY: ci
ci: install-tools tidy lint test coverage
