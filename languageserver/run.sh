#!/bin/sh

SCRIPTPATH=$(dirname "$0")


if [ "$1" = "cadence" ] && [ "$2" = "language-server" ] ; then
	(cd "$SCRIPTPATH" && go build -gcflags="all=-N -l" ./cmd/languageserver && ./languageserver "$@");
else
	flow "$@"
fi
