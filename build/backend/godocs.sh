#!/usr/bin/env bash
scriptpath=$(realpath "$0")
backendpath=$(dirname "$scriptpath")
buildpath=$(dirname "$backendpath")
mainpath=$(dirname "$buildpath")

input=$mainpath
if [ ! -d "$input" ]; then
    echo "Error: Input '$input' is not a directory"
    exit 2
fi

exec docker run --rm -i -t --user="$(id -u):$(id -g)" -p 127.0.0.1:6060:6060 -v "$input":/data blang/posty-build-backend bash -c "GOPATH=\$(wgo env GOPATH) godoc -http ':6060'"
