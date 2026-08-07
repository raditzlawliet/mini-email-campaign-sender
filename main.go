package main

import (
	"embed"
	"log/slog"
	"os"

	"github.com/raditzlawliet/test-mass-email/internal/app"
	"github.com/raditzlawliet/test-mass-email/internal/config"
	"github.com/raditzlawliet/test-mass-email/internal/store"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

var version = "dev"

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	configPath := config.ResolveConfigPath()

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	store.InitStore()
	st := store.GetStore()

	application := app.NewApp(cfg, st, configPath, version)

	slog.Info("starting MECS - Mini Email Campaign Sender", "version", version, "provider", cfg.Email.Provider)

	err = wails.Run(&options.App{
		Title:     "MECS - Mini Email Campaign Sender",
		Width:     1280,
		Height:    900,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 59, A: 1},
		OnStartup:        application.Startup,
		Bind: []interface{}{
			application,
		},
	})
	if err != nil {
		slog.Error("wails run error", "error", err)
		os.Exit(1)
	}
}
