#!/bin/bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

/usr/lib/$(arch)-linux-gnu/mandos/plugins.d/mandos-client -c 127.0.0.1:8080 -p $SCRIPT_DIR/pubkey.txt -s $SCRIPT_DIR/seckey.txt -t $SCRIPT_DIR/tls-privkey.pem -T $SCRIPT_DIR/tls-pubkey.pem
