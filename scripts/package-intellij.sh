#!/usr/bin/env bash
# builds the intellij plugin with every platform's pgfmt binary bundled, from
# the release archives in a dist folder. the plugin zip is written to the dist folder.
# usage: scripts/package-intellij.sh <version> <dist-dir>
set -euo pipefail

version="${1#v}"
dist="$(cd "$2" && pwd)"
plugin="$(cd "$(dirname "$0")/../editors/intellij" && pwd)"

rm -rf "$plugin/bin"
for target in darwin-arm64 darwin-amd64 linux-amd64 linux-arm64 windows-amd64; do
  mkdir -p "$plugin/bin/$target"
  if [ -f "$dist/pgfmt-$target.zip" ]; then
    unzip -q -o "$dist/pgfmt-$target.zip" pgfmt.exe -d "$plugin/bin/$target"
  else
    tar -xzf "$dist/pgfmt-$target.tar.gz" -C "$plugin/bin/$target" ./pgfmt
    chmod +x "$plugin/bin/$target/pgfmt"
  fi
done
cd "$plugin"
./gradlew buildPlugin -PpluginVersion="$version" --console=plain -q
cp build/distributions/pgfmt-intellij-"$version".zip "$dist/"
echo "built $dist/pgfmt-intellij-$version.zip"
