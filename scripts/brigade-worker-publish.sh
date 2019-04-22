#!/bin/sh

function dumpDockerdLogs {
  set +x
  echo "---------- Dumping dockerd logs ----------"
  cat dockerd.logs
}

trap dumpDockerdLogs EXIT

set -euox pipefail

dockerd-entrypoint.sh &> dockerd.logs &

sleep 5

docker build . -t lovethedrake/prototype-brigade-worker:edge
docker push lovethedrake/prototype-brigade-worker:edge
