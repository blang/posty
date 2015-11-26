#!/usr/bin/env bash
scriptpath=$(realpath "$0")
stagingpath=$(dirname "$scriptpath")
buildpath=$(dirname "$stagingpath")
mainpath=$(dirname "$buildpath")

dockerfilepath="$stagingpath/Dockerfile"
cd "$mainpath"
docker build -t blang/posty-staging -f "$dockerfilepath" .
