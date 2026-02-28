#!/bin/bash

set -e
set +x

# parameters
export version=$1
export arch=$2
export binfile=$3
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# constants
if [[ -z "$BUILD_DIR" ]]; then
    BUILD_DIR=/tmp/build
fi

# prepare build dir
rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR

# prepare control
mkdir -p "$BUILD_DIR/mandos2matrix/DEBIAN"
envsubst < "$SCRIPT_DIR/control" > "$BUILD_DIR/mandos2matrix/DEBIAN/control"

# prepare data
mkdir -p "$BUILD_DIR/mandos2matrix/usr/bin"
chmod +x "$binfile"
cp "$binfile" "$BUILD_DIR/mandos2matrix/usr/bin/"

# create deb file
cd "$BUILD_DIR"
dpkg-deb --root-owner-group --build mandos2matrix mandos2matrix_"$version"_"$arch".deb
