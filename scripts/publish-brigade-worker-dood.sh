#!/bin/sh

# AVOID INVOKING THIS SCRIPT DIRECTLY -- USE `drake run publish-brigade-worker-dood

set -euo pipefail

source scripts/versioning.sh
source scripts/base-docker.sh

set +x # Don't let the value of $DOCKER_PASSWORD bleed into the logs!
docker login -u krancour -p $DOCKER_PASSWORD

scripts/docker-publish.sh
