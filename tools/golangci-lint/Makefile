D := $(or $(patsubst %/,%,$(dir $(lastword $(MAKEFILE_LIST)))),.)
$(D)/golangci-lint: $(D)/main.go $(D)/go.mod
	cd $(dir $@) && \
	go build -o golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint
