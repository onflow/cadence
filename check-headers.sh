#!/bin/sh

files=$(find . -name \*.go -type f -print0 | xargs -0 grep -L -E '(Licensed under the Apache License)|(Code generated (from|by))|(Copyright .*\. All rights reserved)')
if [ -n "$files" ]; then
  echo "Missing license header in:"
  echo "$files"
  exit 1
fi
