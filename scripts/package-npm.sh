#!/usr/bin/env bash
# builds the npm packages from the release archives in a dist folder: one
# package per platform holding its binary, and the @sirmspencer/pgfmt package
# that installs the matching one and runs it. npm rejects the unscoped name pgfmt
# as too similar to an existing package, so every package is scoped. output goes to <dist-dir>/npm/<package>.
# usage: scripts/package-npm.sh <version> <dist-dir>
set -euo pipefail

version="${1#v}"
dist="$(cd "$2" && pwd)"
root="$(cd "$(dirname "$0")/.." && pwd)"
out="$dist/npm"
rm -rf "$out" && mkdir -p "$out"

# release archive name -> npm os and cpu
targets="darwin-arm64:darwin:arm64 darwin-amd64:darwin:x64 linux-amd64:linux:x64 linux-arm64:linux:arm64 windows-amd64:win32:x64"
repository='{"type": "git", "url": "git+https://github.com/sirmspencer/pgfmt.git"}'
optional=""

for triple in $targets; do
  IFS=: read -r archive os cpu <<<"$triple"
  name="@sirmspencer/pgfmt-$os-$cpu"
  dir="$out/pgfmt-$os-$cpu"
  mkdir -p "$dir"
  if [ -f "$dist/pgfmt-$archive.zip" ]; then
    unzip -q -o "$dist/pgfmt-$archive.zip" pgfmt.exe -d "$dir"
    file="pgfmt.exe"
  elif [ -f "$dist/pgfmt-$archive.tar.gz" ]; then
    tar -xzf "$dist/pgfmt-$archive.tar.gz" -C "$dir" ./pgfmt
    chmod +x "$dir/pgfmt"
    file="pgfmt"
  else
    echo "missing archive for $archive" >&2
    exit 1
  fi
  cp "$root/LICENSE" "$dir/"
  cat > "$dir/package.json" <<JSON
{
  "name": "$name",
  "version": "$version",
  "description": "The pgfmt binary for $os $cpu",
  "license": "MIT",
  "repository": $repository,
  "os": ["$os"],
  "cpu": ["$cpu"],
  "files": ["$file", "LICENSE"]
}
JSON
  optional="$optional\"$name\": \"$version\","
done

dir="$out/pgfmt"
cp -R "$root/npm/pgfmt/." "$dir"
cp "$root/LICENSE" "$root/README.md" "$dir/"
cat > "$dir/package.json" <<JSON
{
  "name": "@sirmspencer/pgfmt",
  "version": "$version",
  "description": "Formatter for PostgreSQL sql files",
  "license": "MIT",
  "repository": $repository,
  "homepage": "https://github.com/sirmspencer/pgfmt",
  "keywords": ["sql", "postgresql", "postgres", "formatter"],
  "bin": { "pgfmt": "bin/pgfmt.js" },
  "files": ["bin", "LICENSE", "README.md"],
  "optionalDependencies": { ${optional%,} }
}
JSON
echo "packages in $out:"
ls "$out"
