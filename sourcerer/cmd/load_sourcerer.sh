#!/bin/bash

pushd . > /dev/null
cd "$(dirname -- "${BASH_SOURCE[0]}")/sourcerer"
tmpFile="$(mktemp)"
go run *.go sourcerer > $tmpFile && source $tmpFile
popd > /dev/null

# Below are help functions and aliases
function gg {
  for package in "$@"
  do
    if [ "$GO111MODULE" == "on" ]; then
      commitSha="$(git ls-remote git@github.com:leep-frog/${package}.git | grep ma[is][nt] | awk '{print $1}')"
      go get -v "github.com/leep-frog/$package@$commitSha"
    else
      go get -u "github.com/leep-frog/$package"
    fi
  done
}

alias u=mancli
