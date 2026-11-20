# Agent instructions

Lounge is a PLUNZ standalone client (`app.lounge`): a Linux TV app for Stremio addons. Shared PLUNZ rules live in the `plunz` workspace repository (its `AGENTS.md` and `decisions/`). This file covers what is specific to this repository.

## Architecture

The app is one Go binary built with Wails 3. The window is GTK 4 with WebKitGTK 6.0, and the UI is SolidJS + TypeScript built by Vite.

- `main.go` wires the stores, addon client, torrent engine, and mpv player. It creates the window and registers the Wails services.
- `services.go` and `player_service.go` define the Wails services the UI calls by name (`main.<Service>.<Method>`). They are thin adapters; the logic lives in `internal/`.
- `internal/player` drives libmpv through cgo:
  - `render.c` moves the Wails webview into a `GtkOverlay` above a `GtkGLArea`;
  - libmpv renders into that `GtkGLArea` through its OpenGL render API;
  - the webview background is transparent, so the UI draws over the video;
  - property changes reach the UI as `player` events with JSON values.
- `internal/stremio` implements the addon protocol:
  - manifests, catalogs, meta, streams, and subtitles;
  - the installed addon list;
  - queries that fan out across the enabled addons.
  - Stremio account calls: `api.strem.io/api/` for login and addonCollectionGet, and `link.stremio.com/api/v2/` for code sign-in. `account_service.go` mirrors the account's addons, read-only, into the local list.
- `internal/torrent` wraps `anacrolix/torrent` and serves files over `127.0.0.1` HTTP with range support, so mpv can stream and seek while pieces download.
- `internal/library` stores watch progress and watched videos. `internal/settings` stores preferences. `internal/store` holds the shared JSON documents in `~/.config/lounge/`.
- `frontend/src`:
  - `nav.ts` provides spatial navigation for remotes: `data-nav`, `data-nav-group`, `data-nav-entry`, `data-nav-row`, `data-nav-pane`, `data-nav-block`, `data-nav-top` (a fraction of the pane, or `nearest`), `data-nav-scope`, and the back stack. It scrolls with its own short animation, because WebKit's smooth scrolling lags behind a held arrow key;
  - `router.ts` is a stack router whose screens stay mounted underneath;
  - `api.ts` holds the typed service calls;
  - `i18n.ts` has the en/pt/es strings;
  - `screens/` holds the screens.

## Conventions

- Write code, comments, commits, and docs in English. Every user-facing string goes through `t()`, with entries in all three dictionaries.
- Focusable UI elements are real `<button>`s or inputs marked `data-nav`. Style focus with the `.focused` class, not `:focus`; the class still shows when the window is not focused.
- Use `rem` for sizes. The root font size scales with the screen height, so layouts hold from 720p to 4K.
- Colors come from the shared Darkppuccin tokens (`--plz-*`) of `@plunz/storybook`. The accent is selectable.
- Keep Wails services thin and put logic in `internal/` packages with Go tests.
- Lumera is a feature and UX reference only. Do not copy or translate its code: it is GPL-3.0 and this repository is not.
- Pin dependencies. Use pnpm only; never npm, yarn, or Bun. `@plunz/storybook` is pinned to a Git revision.

## Commands

```bash
pnpm install          # dependencies
pnpm dev              # app with Vite hot reload (scripts/dev.sh)
pnpm build            # frontend/dist + build/lounge
pnpm check            # tsc, go vet, go test
./scripts/install.sh  # install for the current user
```

Build requirements are Go, pnpm, and the development files for `gtk4`, `webkitgtk-6.0`, `mpv`, and `epoxy`.

## Testing in a real window

UI changes need a check in the real window, not only type checks. On Hyprland you can test without disturbing the user's screen:

1. Create a headless output: `hyprctl output create headless`.
2. Launch the app on its workspace with `hl.dsp.exec_cmd(cmd, { workspace = '<n> silent' })`.
3. Send keys to the window with `hl.dsp.send_key_state({ ..., window = 'pid:<test pid>' })`. Target it by PID: matching by class would also hit an installed Lounge the user has open.
4. Capture the result with `grim -o <output>`.

Use a separate `XDG_CONFIG_HOME` so tests do not change the user's addons or progress.

## Completion checks

- `pnpm check` and `pnpm build` pass.
- The changed flow works in the app window with arrow keys, Enter, and Back.
- New strings exist in en, pt, and es.
- `README.md` and this file still describe the app.
