#!/usr/bin/env bash
# Package a built Lounge (pnpm build) as a release archive:
# build/release/lounge-linux-x86_64.tar.gz, whose fixed name lets
# /releases/latest/download/ link to it, and its SHA-256 sum. The archive
# holds a lounge-<version> directory.
set -euo pipefail

version="${1:?usage: scripts/package.sh <version>}"
repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

name="lounge-linux-x86_64"
stage="build/release/lounge-$version"
rm -rf build/release
mkdir -p "$stage"
install -m755 build/lounge scripts/install.sh "$stage/"
install -m644 assets/nz.plu.lounge.desktop assets/nz.plu.lounge.svg README.md "$stage/"
tar -C build/release -czf "build/release/$name.tar.gz" "lounge-$version"
(cd build/release && sha256sum "$name.tar.gz" > "$name.tar.gz.sha256")
echo "build/release/$name.tar.gz"
