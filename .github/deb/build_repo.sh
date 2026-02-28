#!/bin/bash

REPO_NAME=mandos2matrix
APTLY_CONF="./.github/deb/aptly.conf"
FS_ENDPOINT="filesystem:$REPO_NAME-repo:deb"
#GPG_KEY_ID= must be given in environment

set -eo pipefail

cp "$APTLY_CONF" ~/.aptly.conf

if ! aptly repo show "$REPO_NAME" > /dev/null 2>&1; then
    aptly repo create -distribution="all" "$REPO_NAME"
fi

if aptly publish show \
        "all" \
        "${FS_ENDPOINT}" > /dev/null 2>&1;
    then
        aptly publish drop "all" "${FS_ENDPOINT}"
fi

aptly repo add "$REPO_NAME" ./new-debs/*.deb

aptly publish repo \
    -force-overwrite \
    -gpg-key "$GPG_KEY_ID" \
    "$REPO_NAME" \
    "${FS_ENDPOINT}"