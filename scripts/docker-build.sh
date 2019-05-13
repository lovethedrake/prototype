#!/bin/sh

set -euox pipefail

docker build . -t $base_image_name:$git_version
docker tag $base_image_name:$git_version $base_image_name:$rel_version
