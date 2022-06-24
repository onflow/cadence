#!/bin/sh

# Exit as soon as any command fails
set -e

BASE_URL="https://storage.googleapis.com/flow-cli"
# The version to download, set by get_version (defaults to args[1])
VERSION="$1"
# The architecture string, set by get_architecture
ARCH=""

# Get the architecture (CPU, OS) of the current system as a string.
# Only MacOS/x86_64/ARM64 and Linux/x86_64 architectures are supported.
get_architecture() {
    _ostype="$(uname -s)"
    _cputype="$(uname -m)"
    _targetpath=""
    if [ "$_ostype" = Darwin ] && [ "$_cputype" = i386 ]; then
        if sysctl hw.optional.x86_64 | grep -q ': 1'; then
            _cputype=x86_64
        fi
    fi
    case "$_ostype" in
        Linux)
            _ostype=linux
            _targetpath=$HOME/.local/bin
            ;;
        Darwin)
            _ostype=darwin
            _targetpath=/usr/local/bin
            ;;
        *)
            echo "unrecognized OS type: $_ostype"
            return 1
            ;;
    esac
    case "$_cputype" in
        x86_64 | x86-64 | x64 | amd64)
            _cputype=x86_64
            ;;
         arm64)
            _cputype=arm64
            ;;
        *)
            echo "unknown CPU type: $_cputype"
            return 1
            ;;
    esac
    _arch="${_cputype}-${_ostype}"
    ARCH="${_arch}"
    TARGET_PATH="${_targetpath}"
}

# Get the latest version from remote if none specified in args.
get_version() {
  if [ -z "$VERSION" ]
  then
    VERSION=$(curl -s "$BASE_URL/cadence-analyzer-version.txt")
  fi
}

# Determine the system architecture, download the appropriate binary, and
# install it in `/usr/local/bin` on macOS and `~/.local/bin` on Linux
# with executable permission.
main() {

  get_architecture || exit 1
  get_version || exit 1

  echo "Downloading version $VERSION ..."

  tmpfile=$(mktemp 2>/dev/null || mktemp -t cadence-analyzer)

  url="$BASE_URL/cadence-analyzer-$ARCH-$VERSION"
  curl --progress-bar "$url" -o $tmpfile

  # Ensure we don't receive a not found error as response.
  if grep -q "The specified key does not exist" $tmpfile
  then
    echo "Version $VERSION could not be found"
    exit 1
  fi

  chmod +x $tmpfile

  [ -d $TARGET_PATH ] || mkdir -p $TARGET_PATH
  mv $tmpfile $TARGET_PATH/cadence-analyzer

  echo "Successfully installed the Cadence Analyzer to $TARGET_PATH."
  echo "Make sure $TARGET_PATH is in your \$PATH environment variable."
}

main

