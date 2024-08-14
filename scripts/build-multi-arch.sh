#!/usr/bin/env bash

target="$1"
output_prefix="$2"

platforms=("windows/amd64" "windows/arm" "linux/amd64" "linux/arm64" "linux/arm")

# Ensure that the target and output_prefix are provided
if [ -z "$target" ] || [ -z "$output_prefix" ]; then
    echo "Usage: $0 <target> <output_prefix>"
    exit 1
fi

for platform in "${!platforms[@]}"; do
    output_name="$output_prefix-${platforms[$platform]}"

    if [[ "$platform" == "windows"* ]]; then
        output_name+='.exe'
    fi

    if env GOOS="${platforms[$platform]%%/*}" GOARCH="${platforms[$platform]##*/}" go build -ldflags="-s" -trimpath -o "$output_name" "$target"; then
        echo "Built $output_name"
    else
        echo "Error building $output_name"
        exit 1
    fi
done