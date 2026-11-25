#!/usr/bin/env bash
# builds release archives into dist/ for the given targets, such as darwin-arm64.
# cgo needs a c compiler for each target; macos builds both of its targets with clang.
# usage: scripts/build.sh <version> <os-arch>...
set -euo pipefail

version="$1"
shift
mkdir -p dist

for target in "$@"; do
  os="${target%-*}"
  arch="${target#*-}"
  name="pgfmt"
  cc="${CC:-}"
  if [ "$os" = "darwin" ]; then
    # clang on macos builds either architecture
    cc="clang -arch $([ "$arch" = amd64 ] && echo x86_64 || echo arm64)"
  fi
  if [ "$os" = "windows" ]; then
    name="pgfmt.exe"
  fi
  work="dist/$target"
  mkdir -p "$work"
  CGO_ENABLED=1 GOOS="$os" GOARCH="$arch" CC="${cc:-cc}" \
    go build -trimpath -ldflags "-s -w -X main.version=$version" -o "$work/$name" ./cmd/pgfmt
  cp LICENSE README.md "$work/"
  if [ "$os" = "windows" ]; then
    (cd "$work" && 7z a -tzip "../pgfmt-$target.zip" . >/dev/null)
  else
    tar -czf "dist/pgfmt-$target.tar.gz" -C "$work" .
  fi
  rm -rf "$work"
  echo "built dist/pgfmt-$target"
done
