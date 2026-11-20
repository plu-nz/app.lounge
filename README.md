# Lounge

Lounge is a Linux TV app for watching content from [Stremio addons](https://github.com/Stremio/stremio-addon-sdk) with a remote control, a gamepad, or a keyboard. It shows catalogs, details, episodes, and sources in a 10-foot interface and plays video with mpv inside the same window.

"Lounge" is a working name. The design takes inspiration from [Lumera](https://github.com/LumeraD3v/Lumera), an Android TV client, but Lounge is written from scratch and contains no Lumera code.

## Features

- Install, enable, reorder, and remove Stremio addons by URL. Cinemeta and OpenSubtitles v3 are installed on first launch.
- Sign in with a Stremio account, either with a QR code or code confirmed on a phone, or with email and password. While signed in:
  - the addon list mirrors the account's collection, synced at sign-in, at every launch, and on demand;
  - the list is read-only in Lounge, so the account is never changed;
  - only the auth key is stored, in `~/.config/lounge/account.json`.
- Home, Movies, and Series screens with catalog rows and a backdrop that follows the focused title.
- "View all" grids with genre filters, and search across every searchable catalog.
- Details for movies and series, with seasons, episodes, and watched marks.
- Sources in horizontal rows, one per addon, with quality and size. Movies show them when the title opens; series show them after you pick an episode from the horizontal episode row.
- Direct HTTP streams and BitTorrent streams. Torrents are streamed by an embedded engine, so Stremio Service is not needed.
- In-window playback through libmpv, with controls over the video:
  - seek and pause;
  - audio and subtitle tracks, including addon subtitles;
  - playback speed;
  - switching sources while keeping the position.
- Continue watching with resume. When resuming, Lounge prefers the last source used.
- Automatic next episode, using the same binge group when the addon provides one.
- English, Portuguese, and Spanish UI. The language follows the system unless one is chosen in Settings.

## Controls

| Key | Action |
| --- | --- |
| Arrows | Move focus. In the player with hidden controls, left and right seek 10 s |
| Enter (OK) | Select. In the player with hidden controls, pause and show the controls |
| Menu | In the player, show or hide the controls without pausing |
| Escape, Backspace, Back | Go back. In the player, back first hides the controls |
| Space | Pause or play |
| F11 | Toggle fullscreen |
| Media keys | Play, pause, next episode, seek |

Remote controls that present themselves as keyboards work as they are.

To see which keys a remote sends, run `LOUNGE_DEBUG=1 lounge`: every key press is logged as `ui: key=...`.

## Requirements

Lounge targets Linux with GTK 4 and WebKitGTK 6.0. To build it you need:

- Go 1.27 or later
- pnpm and Node.js
- the development files for `gtk4`, `webkitgtk-6.0`, `mpv` (libmpv), and `epoxy`

On Arch Linux and Omarchy:

```bash
sudo pacman -S --needed go pnpm gtk4 webkitgtk-6.0 mpv libepoxy
```

## Install

```bash
./scripts/install.sh
```

This builds the app and installs:

- `~/.local/bin/lounge`;
- a desktop entry (`nz.plu.lounge.desktop`);
- an icon.

Lounge starts fullscreen. Set `LOUNGE_WINDOWED=1` to start in a window.

## Develop

```bash
pnpm install
pnpm dev        # Vite with hot reload inside the app window
pnpm check      # TypeScript, go vet, and Go tests
pnpm build      # frontend/dist and build/lounge
```

The project layout and conventions are in [AGENTS.md](AGENTS.md).

## Data

- Settings, installed addons, and watch progress are stored as JSON in `~/.config/lounge/`.
- Torrent data is kept in `~/.cache/lounge/torrents/` only while it plays. Lounge removes it when playback stops, at exit, and at the next launch.
- Lounge does not seed.
