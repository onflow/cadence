#!/bin/sh

SCRIPTPATH=$(dirname "$0")

(cd "$SCRIPTPATH" && go run ./main.go)
