#!/bin/sh

SCRIPTPATH=$(dirname "$0")

if [ "$1" = "cadence" ] && [ "$2" = "language-server" ] ; then
	(cd "$SCRIPTPATH" && \
		go build -gcflags="all=-N -l" ./cmd/languageserver && \
		dlv --log-dest 2 --continue --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec ./languageserver "$@");
else
	flow "$@"
fi
