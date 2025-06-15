#!/usr/bin/env bash

script_dir=$(dirname "$0")
cd $script_dir
pwd

go mod tidy -v

export BUILDX_NO_DEFAULT_ATTESTATIONS=1
export LOCALARCH=$(uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')

echo "Running on $LOCALARCH architecture"
docker buildx build --progress=plain --load --platform linux/$LOCALARCH --build-arg LOCALARCH=$LOCALARCH --sbom=false --provenance=false --builder mcp-builder -t dryruntool:v2 .

