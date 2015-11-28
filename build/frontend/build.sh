#!/usr/bin/env bash
scriptpath=$(realpath "$0")
frontendpath=$(dirname "$scriptpath")
buildpath=$(dirname "$frontendpath")
mainpath=$(dirname "$buildpath")
input="$mainpath/frontend"
if [ ! -d "$input" ]; then
    echo "Error: Input '$input' is not a directory"
    exit 2
fi

exec docker run --rm -i -t --user="$(id -u):$(id -g)" -v "$input":/data blang/posty-build-frontend bash -c "npm install && bower install && grunt build"
