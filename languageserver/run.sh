#!/bin/sh

SCRIPTPATH=$(dirname "$0")


if [ "$1" = "cadence" ] && [ "$2" = "language-server" ] ; then
	(cd "$SCRIPTPATH" && /usr/local/bin/go build -gcflags='-N -l' ./cmd/languageserver && ./languageserver "$@");
else
	flow "$@"
fi

