#!/usr/bin/env bash
set -euo pipefail

apt-get update

apt-get install -y --no-install-recommends \
  git \
  make \
  ca-certificates \
  protobuf-compiler \
  pkg-config \
  build-essential \
  libenet-dev

rm -rf /var/lib/apt/lists/*
