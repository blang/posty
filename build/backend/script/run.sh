#!/usr/bin/env bash
set -e -o pipefail
trap 'An error occurred' ERR
mkdir -p /tmp/build
cp -a /data/. /tmp/build/
rm -rf /tmp/build/vendor || true
cd /tmp/build

echo "# Install wgo"
go get -v github.com/skelterjohn/wgo

echo "# Restoring dependencies"
wgo restore 

echo "# Building posty"
wgo build posty

echo "# Build successful, copy data"
chown $USERID:$GROUPID /tmp/build -R
cp -v -a /tmp/build/posty /output/
echo "# Copy data successful"
