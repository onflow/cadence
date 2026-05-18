#!/bin/bash

set -euo pipefail

if [ $# -lt 1 ] || [ -z "$1" ]; then
  echo "Usage: $0 <major|minor|patch|version>"
  exit 1
fi

# Read the current version from version.go (only used for major/minor/patch increments).
v=$(sed -En 's/const Version = "v(.*)"/\1/p' version.go)

case "$1" in
  major)
    v2=$(echo "$v" | awk -F. '{print $1 + 1 ".0.0"}')
    ;;

  minor)
    v2=$(echo "$v" | awk -F. '{print $1 "." $2 + 1 ".0"}')
    ;;

  patch)
    v2=$(echo "$v" | awk -F. '{print $1 "." $2 "." $3 + 1}')
    ;;

  *)
    # Trim off preceding `v` if any.
    # This is to support both input version formats: `0.1.0` and `v0.1.0`.
    v2=$(echo "$1" | sed -Ee 's/^v//')
    ;;
esac

echo "$v => $v2"

# Pick the correct in-place flag for sed on macOS vs. Linux.
if [[ "$OSTYPE" == "darwin"* ]]; then
  SED_INPLACE=(sed -i '' -E)
else
  SED_INPLACE=(sed -i -E)
fi

# version.go: replace `const Version = "..."` regardless of the original content.
echo "- version.go"
"${SED_INPLACE[@]}" "s/const Version = \"[^\"]*\"/const Version = \"v${v2}\"/" version.go
if ! grep -q "const Version = \"v${v2}\"" version.go; then
  echo "failed to update version in version.go"
  exit 1
fi

# npm-packages/cadence-parser/package.json: replace `"version": "..."` regardless of the original content.
echo "- npm-packages/cadence-parser/package.json"
"${SED_INPLACE[@]}" "s/\"version\": \"[^\"]*\"/\"version\": \"${v2}\"/" npm-packages/cadence-parser/package.json
if ! grep -q "\"version\": \"${v2}\"" npm-packages/cadence-parser/package.json; then
  echo "failed to update version in npm-packages/cadence-parser/package.json"
  exit 1
fi
