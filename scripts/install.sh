#!/usr/bin/env bash
# Build Lounge and install it for the current user: the binary in
# ~/.local/bin, plus a desktop entry and icon so launchers list it.
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

pnpm install --frozen-lockfile
pnpm build

bin_dir="$HOME/.local/bin"
data_dir="${XDG_DATA_HOME:-$HOME/.local/share}"
install -Dm755 build/lounge "$bin_dir/lounge"
install -Dm644 assets/nz.plu.lounge.desktop "$data_dir/applications/nz.plu.lounge.desktop"
install -Dm644 assets/nz.plu.lounge.svg "$data_dir/icons/hicolor/scalable/apps/nz.plu.lounge.svg"
update-desktop-database "$data_dir/applications" 2>/dev/null || true
echo "Installed $bin_dir/lounge"
