#!/usr/bin/env bash
set -euo pipefail

apt-get update

apt-get install -y --no-install-recommends \
  git \
  make \
  ca-certificates \
  pkg-config \
  build-essential \
  libenet-dev \
  postgresql-client \
  redis-tools

rm -rf /var/lib/apt/lists/*
