#!/bin/sh
set -eu

if [ "$(uname -s)" = Darwin ]; then
    source_dir=src/infrastructure/dictation/macos
    xcrun swiftc -O "$source_dir/main.swift" \
        -Xlinker -sectcreate -Xlinker __TEXT -Xlinker __info_plist -Xlinker "$source_dir/Info.plist" \
        -o factory-board-dictation
fi
go build -o factory-board ./src
