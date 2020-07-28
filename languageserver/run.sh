#!/bin/sh

SCRIPTPATH=$(dirname "$0")

(cd "$SCRIPTPATH" && go run ./cmd/languageserver/main.go)
