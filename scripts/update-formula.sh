#!/usr/bin/env bash
# writes the homebrew formula for a release, using its prebuilt archives and
# checksums.txt, so brew installs a binary instead of building from source.
# usage: scripts/update-formula.sh <version> <formula-path>
set -euo pipefail

tag="v${1#v}"
version="${tag#v}"
formula="$2"
checksums="$(gh release download "$tag" --repo sirmspencer/pgfmt --pattern checksums.txt --output -)"
sum() { echo "$checksums" | awk -v file="pgfmt-$1.tar.gz" '$2 == file { print $1 }'; }
url="https://github.com/sirmspencer/pgfmt/releases/download/$tag"

cat > "$formula" <<RUBY
class Pgfmt < Formula
  desc "Formatter for PostgreSQL sql files"
  homepage "https://github.com/sirmspencer/pgfmt"
  version "$version"
  license "MIT"

  on_macos do
    on_arm do
      url "$url/pgfmt-darwin-arm64.tar.gz"
      sha256 "$(sum darwin-arm64)"
    end
    on_intel do
      url "$url/pgfmt-darwin-amd64.tar.gz"
      sha256 "$(sum darwin-amd64)"
    end
  end

  on_linux do
    on_arm do
      url "$url/pgfmt-linux-arm64.tar.gz"
      sha256 "$(sum linux-arm64)"
    end
    on_intel do
      url "$url/pgfmt-linux-amd64.tar.gz"
      sha256 "$(sum linux-amd64)"
    end
  end

  def install
    bin.install "pgfmt"
  end

  test do
    assert_match "pgfmt v#{version}", shell_output("#{bin}/pgfmt -version")
    assert_equal "select 1\\n", pipe_output("#{bin}/pgfmt", "SELECT 1")
  end
end
RUBY
echo "wrote $formula"
