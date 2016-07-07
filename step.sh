#!/bin/bash

THIS_SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
export THIS_SCRIPT_DIR=$THIS_SCRIPT_DIR

export GO15VENDOREXPERIMENT="1"
GOPATH="$THIS_SCRIPT_DIR/go" go run "${THIS_SCRIPT_DIR}/go/src/github.com/bitrise-io/install-missing-android-tools/main.go"
exit $?