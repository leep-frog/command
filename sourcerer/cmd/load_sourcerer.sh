#!/bin/bash

pushd . > /dev/null
cd "$(dirname -- "${BASH_SOURCE[0]}")/sourcerer"
source "$(go run *.go sourcerer)"
popd > /dev/null
