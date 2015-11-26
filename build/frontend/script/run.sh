#!/usr/bin/env bash
trap 'An error occurred' ERR
mkdir /tmp/build
cp -a /data/. /tmp/build/

cd /data

echo "# Install basics"
npm install -g grunt-cli bower

echo "# Install deps"
npm install

echo "# Build"
grunt build

echo "# Build successful copy data"
cp -v -a /tmp/build/dist/. /output/

echo "# Copy data successful"
