#!/usr/bin/env bash
trap 'An error occurred' ERR

docker build -t blang/posty-build-frontend .
