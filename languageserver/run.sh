#!/bin/sh

SCRIPTPATH=$(dirname "$0")


if [ $1 == "cadence" -a $2 == "language-server" ] ; then
	(cd "$SCRIPTPATH" && /usr/local/bin/go run ./cmd/languageserver/main.go "$@");
else
	flow "$@"
fi

