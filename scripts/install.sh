#!/usr/bin/env bash
# Install Lounge for the current user: the binary in ~/.local/bin, plus a
# desktop entry and icon so launchers list it. Inside a release archive it
# installs the prebuilt binary; in the repository it builds first.
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -x "$script_dir/lounge" ]]; then
  binary="$script_dir/lounge"
  assets="$script_dir"
else
  repo_dir="$(cd "$script_dir/.." && pwd)"
  cd "$repo_dir"
  pnpm install --frozen-lockfile
  pnpm build
  binary="$repo_dir/build/lounge"
  assets="$repo_dir/assets"
fi

bin_dir="$HOME/.local/bin"
data_dir="${XDG_DATA_HOME:-$HOME/.local/share}"
install -Dm755 "$binary" "$bin_dir/lounge"
install -Dm644 "$assets/nz.plu.lounge.desktop" "$data_dir/applications/nz.plu.lounge.desktop"
install -Dm644 "$assets/nz.plu.lounge.svg" "$data_dir/icons/hicolor/scalable/apps/nz.plu.lounge.svg"
update-desktop-database "$data_dir/applications" 2>/dev/null || true
echo "Installed $bin_dir/lounge"
