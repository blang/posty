#!/usr/bin/env bash
docker run --rm -i -t -v "$1":/data:ro "$2":/output blang/posty-build-frontend
