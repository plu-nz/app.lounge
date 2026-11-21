package main

import (
	"context"
	"embed"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/plu-nz/app.lounge/internal/gtktheme"
	"github.com/plu-nz/app.lounge/internal/library"
	"github.com/plu-nz/app.lounge/internal/player"
	"github.com/plu-nz/app.lounge/internal/settings"
	"github.com/plu-nz/app.lounge/internal/store"
	"github.com/plu-nz/app.lounge/internal/stremio"
	"github.com/plu-nz/app.lounge/internal/torrent"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

const appID = "nz.plu.lounge"

func main() {
	// GTK's default Vulkan renderer has to import every OpenGL video frame
	// from the GtkGLArea, which halves the frame rate on integrated GPUs.
	// Its OpenGL renderer composites the GL area directly.
	if os.Getenv("GSK_RENDERER") == "" {
		os.Setenv("GSK_RENDERER", "ngl")
	}
	st, err := store.Open("lounge")
	if err != nil {
		log.Fatalf("config dir: %v", err)
	}
	prefs, err := settings.Load(st)
	if err != nil {
		log.Fatalf("settings: %v", err)
	}
	lib, err := library.Load(st)
	if err != nil {
		log.Fatalf("library: %v", err)
	}
	client := stremio.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	addons, err := stremio.LoadAddons(ctx, client, st)
	cancel()
	if err != nil {
		log.Fatalf("addons: %v", err)
	}
	account, err := newAccountService(client, addons, st)
	if err != nil {
		log.Fatalf("account: %v", err)
	}
	cacheDir, _ := os.UserCacheDir()
	torrents, err := torrent.Start(filepath.Join(cacheDir, "lounge"))
	if err != nil {
		// Direct HTTP streams still work without the torrent engine.
		log.Printf("torrents disabled: %v", err)
	}

	phoneAdd := &PhoneAddService{addons: addons, client: client}
	themes := &ThemeService{debug: os.Getenv("LOUNGE_DEBUG") != ""}
	var app *application.App
	video, err := player.New(func(e player.Event) {
		if app != nil {
			app.Event.Emit("player", e)
		}
	})
	if err != nil {
		log.Fatalf("start mpv: %v", err)
	}

	title, id := "Lounge", appID
	if devRun() {
		title, id = "Lounge (dev)", appID+".dev"
	}
	app = application.New(application.Options{
		Name:  "Lounge",
		Linux: application.LinuxOptions{ApplicationID: id},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Services: []application.Service{
			application.NewService(&PlayerService{player: video, torrents: torrents}),
			application.NewService(&ContentService{content: &stremio.Content{Client: client, Addons: addons}}),
			application.NewService(&AddonService{addons: addons}),
			application.NewService(phoneAdd),
			application.NewService(&LibraryService{library: lib}),
			application.NewService(&SettingsService{settings: prefs}),
			application.NewService(newSegmentService()),
			application.NewService(themes),
			application.NewService(account),
			application.NewService(&AppService{debug: os.Getenv("LOUNGE_DEBUG") != ""}),
		},
		OnShutdown: func() {
			application.InvokeSync(video.Detach)
			video.Close()
			if torrents != nil {
				torrents.Close()
			}
		},
	})

	phoneAdd.emit = func(name string, data ...any) { app.Event.Emit(name, data...) }
	// A signed-in account's addons are mirrored on every launch.
	account.changed = func() { app.Event.Emit("addons") }
	if account.Status().SignedIn {
		go func() {
			if _, err := account.Sync(); err != nil {
				log.Printf("stremio sync: %v", err)
			}
		}()
	}

	start := application.WindowStateFullscreen
	if devRun() || os.Getenv("LOUNGE_WINDOWED") != "" {
		start = application.WindowStateNormal
	}
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            title,
		Width:            1280,
		Height:           720,
		MinWidth:         960,
		MinHeight:        540,
		StartState:       start,
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		DevToolsEnabled:  devRun(),
		// The remote's Menu key would open WebKit's right-click menu; the
		// UI uses that key itself.
		DefaultContextMenuDisabled: !devRun(),
	})
	// The runtime becomes ready after every page load; attaching is idempotent.
	window.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		application.InvokeSync(func() {
			if err := video.Attach(window.NativeWindow()); err != nil {
				log.Printf("video surface: %v", err)
			}
			// The UI may ask before this runs; the event makes it ask again.
			if themes.set(gtktheme.Read(window.NativeWindow())) {
				app.Event.Emit("theme")
			}
			// GTK restyles after it announces a theme change, so read the
			// colors a moment later.
			gtktheme.Watch(func() {
				time.AfterFunc(200*time.Millisecond, func() {
					changed := application.InvokeSyncWithResult(func() bool {
						return themes.set(gtktheme.Read(window.NativeWindow()))
					})
					if changed {
						app.Event.Emit("theme")
					}
				})
			})
		})
	})

	// Quit cleanly on SIGTERM/SIGINT so torrent data is removed.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sig
		app.Quit()
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// devRun reports a development run started by scripts/dev.sh.
func devRun() bool { return os.Getenv("FRONTEND_DEVSERVER_URL") != "" }
