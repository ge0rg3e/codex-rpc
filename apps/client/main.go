package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

//go:embed build/trayicon.png
var trayIcon []byte

// currentVersion is set from the release tag by the packaging tasks.
var currentVersion = "0.0.3"

//go:embed updater.key.pub
var updaterPublicKey []byte

func main() {
	show := flag.Bool("show", false, "Show the tray popup on startup")
	flag.Parse()
	service, err := NewPresenceService()
	if err != nil {
		log.Fatal(err)
	}
	app := application.New(application.Options{
		Name:        "codex-rpc",
		Icon:        appIcon,
		Description: "A quiet tray companion for Codex",
		Services:    []application.Service{application.NewService(service)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Windows:     application.WindowsOptions{WebviewUserDataPath: filepath.Dir(service.path)},
		Mac:         application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: false},
		Linux:       application.LinuxOptions{ProgramName: "codex-rpc"},
	})
	nativeLinux := runtime.GOOS == "linux" && os.Getenv("APPIMAGE") == ""
	updatesEnabled := true
	{
		var updateWindow updater.WindowOption = updater.WindowNone
		if !nativeLinux {
			updateWindow = &updater.BuiltinWindow{Options: updater.WindowOptions{Title: "codex-rpc update"}}
		}
		provider, err := endpoint.New(endpoint.Config{URL: "https://github.com/ge0rg3e/codex-rpc/releases/latest/download/manifest.json"})
		if err != nil {
			log.Printf("update provider: %v", err)
			updatesEnabled = false
		} else if err := app.Updater.Init(updater.Config{
			CurrentVersion: currentVersion,
			Providers:      []updater.Provider{provider},
			PublicKey:      updaterPublicKey,
			Window:         updateWindow,
		}); err != nil {
			log.Printf("update initialization: %v", err)
			updatesEnabled = false
		}
	}
	service.openURL = app.Browser.OpenURL
	service.quit = app.Quit
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{Title: "codex-rpc", Width: 340, Height: 560, Frameless: true, AlwaysOnTop: true, Hidden: !*show, HideOnFocusLost: !*show, DisableResize: true, BackgroundColour: application.NewRGB(12, 14, 18), URL: "/", Windows: application.WindowsWindow{HiddenOnTaskbar: !*show}})
	tray := app.SystemTray.New()
	tray.SetTooltip("codex-rpc")
	tray.SetIcon(trayIcon)
	tray.AttachWindow(window).WindowOffset(8)
	configureTrayPosition(app, tray, window)
	menu := app.NewMenu()
	if updatesEnabled && !nativeLinux {
		menu.Add("Check for updates").OnClick(func(*application.Context) {
			go func() {
				if err := app.Updater.CheckAndInstall(context.Background()); err != nil {
					log.Printf("update check: %v", err)
				}
			}()
		})
		app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
			go checkForUpdate(app)
		})
		go func() {
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			for range ticker.C {
				checkForUpdate(app)
			}
		}()
	} else if nativeLinux {
		menu.Add("Check for updates").OnClick(func(*application.Context) {
			go showPackageUpdate(app)
		})
	}
	menu.AddSeparator()
	menu.Add("Open leaderboard").OnClick(func(*application.Context) { _ = service.OpenLeaderboard() })
	menu.AddSeparator()
	menu.Add("Quit codex-rpc").OnClick(func(*application.Context) { app.Quit() })
	tray.SetMenu(menu)
	if *show {
		app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) { window.Show().Focus() })
	}
	service.run()
	if err := app.Run(); err != nil {
		log.Print(err)
	}
	close(service.stop)
	<-service.done
}

func checkForUpdate(app *application.App) {
	ctx := context.Background()
	release, err := app.Updater.Check(ctx)
	if err != nil {
		log.Printf("automatic update check: %v", err)
		return
	}
	if release == nil {
		return
	}
	if err := app.Updater.CheckAndInstall(ctx); err != nil {
		log.Printf("automatic update install: %v", err)
	}
}

func showPackageUpdate(app *application.App) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	release, err := app.Updater.Check(ctx)
	message := "You have the latest version of codex-rpc."
	if err != nil {
		log.Printf("package update check: %v", err)
		message = "Could not check for updates. Try again in a moment."
	} else if release != nil {
		message = fmt.Sprintf("codex-rpc %s is available. Run the Linux setup command again to install the update through your package manager.", release.Version)
	}
	application.InvokeAsync(func() {
		app.Dialog.Info().SetTitle("codex-rpc updates").SetMessage(message).Show()
	})
}
