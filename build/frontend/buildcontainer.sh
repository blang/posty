#!/usr/bin/env bash
scriptpath=$(realpath "$0")
frontendpath=$(dirname "$scriptpath")
cd "$frontendpath"
exec docker build -t blang/posty-build-frontend .
