#!/bin/bash

pushd . > /dev/null
cd "$(dirname -- "${BASH_SOURCE[0]}")/sourcerer"
tmpFile="$(mktemp)"
go run . sourcerer > $tmpFile && source $tmpFile
popd > /dev/null

alias u=mancli
