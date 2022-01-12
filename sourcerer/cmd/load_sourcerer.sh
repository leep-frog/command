#!/bin/bash

pushd . > /dev/null
cd "$(dirname -- "${BASH_SOURCE[0]}")/sourcerer"
tmpFile="$(mktemp)"
go run *.go sourcerer > $tmpFile && source $tmpFile
popd > /dev/null
