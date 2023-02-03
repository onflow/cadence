#!/bin/bash

# Read the version to be replaced from the `version.go` file.
v=$(sed -En 's/const Version = "v(.*)"/\1/p' version.go)

case "$1" in
  major)
    v2=$(echo "$v" | awk -F. '{print $1 + 1 ".0.0"}')
    ;;

  minor)
    v2=$(echo "$v" | awk -F. '{print $1 "." $2 + 1  ".0"}')
    ;;

  patch)
    v2=$(echo "$v" | awk -F. '{print $1 "." $2 "." $3 + 1}')
    ;;

  *)

    # Trim off preceding `v` if any.
    # This is to support both input version formats: `0.1.0` and `v0.1.0`.
    v2=$(echo "$1" | sed -Ee 's/^v|-.*//')
    ;;
esac

echo "$v => $v2"

for f in $VERSIONED_FILES; do \
  echo "- $f"; \
  if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "s/$v/$v2/g" "$f"; \
  else
    sed -i "s/$v/$v2/g" "$f"; \
  fi
done
