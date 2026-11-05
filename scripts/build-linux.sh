#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export CGO_ENABLED=0
export GOOS=linux
mkdir -p dist
# Build both tunnel roles into each executable. GOARM=7 is for 32-bit ARMv7.
for ping_arch in amd64 arm64 arm; do
  if [ "$ping_arch" = arm ]; then
    GOARCH=arm GOARM=7 go build -buildvcs=false -mod=readonly -trimpath -o dist/pingtunnel-linux-armv7 ./cmd
  else
    GOARCH="$ping_arch" go build -buildvcs=false -mod=readonly -trimpath -o "dist/pingtunnel-linux-$ping_arch" ./cmd
  fi
done
(cd dist && sha256sum pingtunnel-linux-amd64 pingtunnel-linux-arm64 pingtunnel-linux-armv7 > SHA256SUMS)
