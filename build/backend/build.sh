#!/usr/bin/env bash
scriptpath=$(realpath "$0")
backendpath=$(dirname "$scriptpath")
buildpath=$(dirname "$backendpath")
mainpath=$(dirname "$buildpath")

input=$mainpath
output=$mainpath
if [ ! -d "$input" ]; then
    echo "Error: Input '$input' is not a directory"
    exit 2
fi
if [ ! -d "$output" ]; then
    echo "Error: Output '$output' is not a directory"
    exit 2
fi

exec docker run  -e USERID="$(id -u)" -e GROUPID="$(id -g)" -v "$input":/data:ro -v "$output":/output blang/posty-build-backend