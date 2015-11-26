#!/usr/bin/env bash
scriptpath=$(realpath "$0")
backendpath=$(dirname "$scriptpath")
cd "$backendpath"
exec docker build -t blang/posty-build-backend .
