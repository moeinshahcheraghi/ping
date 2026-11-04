#!/usr/bin/env bash
set -euo pipefail
cd /workspace/ping
ping_dev_root=/workspace/.ping-dev
ping_go_root="$ping_dev_root/toolchains/go1.27.3"
if ! test -x "$ping_go_root/go/bin/go" || ! "$ping_go_root/go/bin/go" version | grep -q 'go1.27.3 linux/amd64'; then
  ping_go_tmp=$(mktemp -d /tmp/ping-go-install.XXXXXX)
  trap 'rm -rf "$ping_go_tmp"' EXIT
  curl --fail --location --retry 2 https://go.dev/dl/go1.27.3.linux-amd64.tar.gz -o "$ping_go_tmp/go.tar.gz"
  printf '%s  %s\n' 65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40 "$ping_go_tmp/go.tar.gz" | sha256sum --check --status
  mkdir -p "$ping_go_root"
  tar -xzf "$ping_go_tmp/go.tar.gz" -C "$ping_go_root"
fi
mkdir -p "$ping_dev_root"
cat > "$ping_dev_root/env.sh" <<'PING_ENV'
export PATH=/workspace/.ping-dev/toolchains/go1.27.3/go/bin:$PATH
export GOTOOLCHAIN=local
export GOPATH=/workspace/.ping-dev/gopath
export GOCACHE=/workspace/.ping-dev/go-build
export GOFLAGS=-mod=readonly
export GOMAXPROCS=4
PING_ENV
source "$ping_dev_root/env.sh"
mkdir -p "$ping_dev_root/bin" "$ping_dev_root/results"
go mod download -modcacherw
go mod verify
CGO_ENABLED=0 go build -p 4 -trimpath -o "$ping_dev_root/bin/pingtunnel-echo" ./cmd
"$ping_dev_root/bin/pingtunnel-echo" -version
