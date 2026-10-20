package main

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"chitchatbot/internal/applog"
	"chitchatbot/internal/runtimeassets"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:cmd/chitchatui/frontend/web
var embeddedAssets embed.FS

func main() {
	dataDir, err := resolveDataDir()
	if err != nil {
		log.Fatal(err)
	}

	// Set up diagnostics before database/license/UI startup, so startup failures
	// from log.Fatal are also retained. Existing console output stays available.
	diskLogger := applog.Logger{Path: applog.Path(dataDir), Source: "wails"}
	// File first: a Windows GUI build may have an unavailable stderr handle.
	log.SetOutput(io.MultiWriter(applog.Logger{Path: diskLogger.Path, Source: "go"}, os.Stderr))
	log.SetFlags(0)
	diskLogger.Info("DESKTOP START | diagnostic logging initialized")
	capture, captureErr := applog.CaptureOutput(diskLogger.Path)
	if captureErr != nil {
		diskLogger.Error("Process diagnostics capture unavailable: " + captureErr.Error())
	} else {
		defer func() {
			if err := capture.Close(); err != nil {
				diskLogger.Error("Diagnostic stream close failed: " + err.Error())
			}
		}()
	}
	defer diskLogger.Info("DESKTOP STOP | main returned")

	dist, err := fs.Sub(embeddedAssets, "cmd/chitchatui/frontend/web")
	if err != nil {
		log.Fatal(err)
	}

	app, err := NewApp(dataDir)
	if err != nil {
		fatalStartup(err, "The data folder could not be opened. Another ChitChat Bot window (or a test tool) may still be running — close it and start again.")
	}

	// Keep the WebView2 profile inside this copy's runtime data. The default
	// (%APPDATA%\<exe name>) is shared by every copy of chitchatui.exe, so a
	// second running copy fails with 0x800700AA ("resource is in use").
	webviewDir := filepath.Join(dataDir, "webview")
	if err := os.MkdirAll(webviewDir, 0o755); err != nil {
		fatalStartup(err, "The data folder is not writable. Move the app to a normal folder (not Program Files) and try again.")
	}

	err = wails.Run(&options.App{
		Title:  "ChitChat Bot",
		Logger: diskLogger,
		// Include framework info/warnings; do not enable raw RPC debug dumps that
		// could serialize settings/credentials. App logs have no severity filter.
		LogLevelProduction: logger.INFO,
		Width:              1280,
		Height:             820,
		MinWidth:           1080,
		MinHeight:          700,
		BackgroundColour:   &options.RGBA{R: 22, G: 23, B: 22, A: 255},
		AssetServer: &assetserver.Options{
			Assets: dist,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			Theme:               windows.SystemDefault,
			WebviewBrowserPath:  runtimeassets.BundledWebView2Path(),
			WebviewUserDataPath: webviewDir,
		},
	})
	if err != nil {
		fatalStartup(err, "The window could not be created. Install Microsoft Edge WebView2 from https://developer.microsoft.com/microsoft-edge/webview2/ and try again.")
	}
}

// fatalStartup turns a silent GUI death into something the owner can read: the
// error goes to startup-error.log next to the exe and into a native dialog.
func fatalStartup(err error, hint string) {
	if err == nil {
		return
	}
	message := err.Error()
	if hint != "" {
		message += "\n\n" + hint
	}
	exe, exeErr := os.Executable()
	if exeErr != nil {
		exe = "."
	}
	logPath := writeStartupError(filepath.Dir(exe), message)
	log.Print(message)
	notifyStartupFailure("ChitChat Bot could not start", message, logPath)
	os.Exit(1)
}

func resolveDataDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, "data"), nil
}
