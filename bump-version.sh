#!/bin/bash

v=$(git describe --tags --abbrev=0 | sed -Ee 's/^v|-.*//')

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
    v2=$1
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
