#!/usr/bin/env bash

set -ex

docker build  --network=host --progress=plain  -t s3tests .
