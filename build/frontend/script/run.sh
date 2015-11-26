#!/usr/bin/env bash
set -e -o pipefail
trap 'An error occurred' ERR
mkdir -p /tmp/build
cp -v -a /data/. /tmp/build/
rm -rf /tmp/build/dist || true
cd /tmp/build

echo "# Install basics"
npm install -g grunt-cli bower

echo "# Install deps"
npm install

echo "# Build"
grunt build

echo "# Build successful copy data"
chown $USERID:$GROUPID /tmp/build/dist -R
cp -v -a /tmp/build/dist/. /output/
echo "# Copy data successful"
