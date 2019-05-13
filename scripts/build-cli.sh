#!/usr/bin/env bash

# AVOID INVOKING THIS SCRIPT DIRECTLY -- USE `drake run build-cli-<os>-<arch>`

set -euo pipefail

goos=$1
if [ "$goos" == "windows" ]; then
  file_ext=".exe"
else
  file_ext=""
fi

goarch=$2

source scripts/versioning.sh

base_package_name=github.com/lovethedrake/prototype
ldflags="-w -X $base_package_name/pkg/version.version=$rel_version -X $base_package_name/pkg/version.commit=$git_version"

set -x

GOOS=$goos GOARCH=$goarch packr2 build -ldflags "$ldflags" -o bin/cli/drake-$goos-$goarch$file_ext ./cmd/cli
