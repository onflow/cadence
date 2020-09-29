#!/bin/sh

SCRIPTPATH=$(dirname "$0")

(cd "$SCRIPTPATH" && /usr/local/bin/go run ./cmd/languageserver/main.go "$@")
