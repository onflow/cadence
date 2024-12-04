#!/bin/zsh

time swift build -c release

# find .build/release/ -type f | sort | grep -v 'o$' | grep -v 'json$' | grep -v 'yml$' | grep -v 'plist$' | grep -v 'DWARF' | grep -v 'LinkFileList'| grep -v 'ModuleCache' | grep -v 'swiftdoc' | grep -v 'swiftmodule' | grep -v '\.d$' | grep -v 'swiftdeps' | grep -v 'modulemap' | grep -v '\h$'

#strip -x -S .build/release/cadence-tool


