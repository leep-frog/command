#!/bin/bash

# Need to wrap in a function so the `source $tmpFile`
# command is run in a function (for local variables).
function _initial_load() {
  pushd . > /dev/null
  cd "$(dirname -- "${BASH_SOURCE[0]}")"
  tmpFile="$(mktemp)"
  go run . sourcerer > $tmpFile && source $tmpFile
  popd > /dev/null
}
_initial_load

aliaser u mancli
