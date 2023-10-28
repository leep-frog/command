#!/bin/bash

# Need to wrap in a function so the `source $tmpFile`
# command is run in a function (for local variables).
function _initial_load() {
  pushd . > /dev/null
  cd "$(dirname -- "${BASH_SOURCE[0]}")"
  tmpFile="$(mktemp)"
  go run . source sourcerer > $tmpFile && source $tmpFile
  popd > /dev/null
}
_initial_load

aliaser mc mancli

# On pre-commit, build a new executable file
# Executable file (generated from `go build -o sourcerer.exe/bin`) will be used to run the following:
# tmpFile="$(mktemp)"
# sourcerer.exe/bin > $tmpFile && source $tmpFile
