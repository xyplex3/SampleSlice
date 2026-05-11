#!/bin/bash

set -e

VERSION="${VERSION:-0.0.0}"
COMMIT="${COMMIT:-unknown}"
DATE="${DATE:-$(date -u +'%Y-%m-%dT%H:%M:%SZ')}"

LDFLAGS="-X sampleslice/cmd.Version=${VERSION} -X sampleslice/cmd.Commit=${COMMIT} -X sampleslice/cmd.BuildDate=${DATE}"

echo "Building sampleslice v${VERSION} (commit: ${COMMIT})"

go build -ldflags "$LDFLAGS" -o sampleslice .

echo "Build successful: ./sampleslice"