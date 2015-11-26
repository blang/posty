#!/bin/bash

env
echo PWD: $PWD

echo "Realpath"
dir=$(realpath "$0")
dir=$(dirname "$dir")
echo Where i live: $dir

scriptpath=$(realpath "$0")
buildpath=$(dirname "$scriptpath")
mainpath=$(dirname "$buildpath")
echo Ok $scriptpath, $buildpath, $mainpath

