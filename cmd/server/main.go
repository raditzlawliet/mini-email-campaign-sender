package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/raditzlawliet/test-mass-email/internal/config"
	"github.com/raditzlawliet/test-mass-email/internal/handler"
	"github.com/raditzlawliet/test-mass-email/internal/store"
)

var version = "dev"

//go:embed frontend_dist/*
var frontendDist embed.FS

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Info("starting MECS - Mini Email Campaign Sender", "version", version, "port", cfg.Server.Port, "provider", cfg.Email.Provider)

	store.InitStore()
	st := store.GetStore()

	h := handler.NewHandler(cfg, st, configPath)

	app := fiber.New(fiber.Config{
		AppName:     "Mini Email Campaign Sender",
		ReadTimeout: 10 * 60 * time.Second,
		BodyLimit:   1024 * 1024 * 1024, // 1GB
	})

	devMode := os.Getenv("DEV_MODE") == "true"

	if devMode {
		app.Use(cors.New(cors.Config{
			AllowOrigins: []string{"http://localhost:5173"},
			AllowMethods: []string{fiber.MethodGet, fiber.MethodPost, fiber.MethodPut, fiber.MethodDelete, fiber.MethodOptions},
			AllowHeaders: []string{"Content-Type"},
		}))
		slog.Info("DEV_MODE enabled — API only, CORS allowed for http://localhost:5173")
	}

	// API routes
	app.Get("/api/version", func(c fiber.Ctx) error {
		return c.JSON(map[string]string{"version": version})
	})
	app.Get("/api/campaign/config", h.HandleGetConfig)
	app.Post("/api/campaign/preview", h.HandlePreview)
	app.Post("/api/campaign/start", h.HandleStart)
	app.Post("/api/campaign/pause", h.HandlePause)
	app.Post("/api/campaign/resume", h.HandleResume)
	app.Get("/api/campaign/events", h.HandleEvents)
	app.Post("/api/campaign/reset", h.HandleReset)
	app.Post("/api/config/save", h.HandleSaveConfig)

	if !devMode {
		app.Use(static.New("", static.Config{
			FS:         frontendFS(),
			Browse:     false,
			IndexNames: []string{"index.html"},
			MaxAge:     3600,
		}))
	}

	// Graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		sig := <-quit
		slog.Info("received shutdown signal", "signal", sig.String())

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := app.ShutdownWithContext(shutdownCtx); err != nil {
			slog.Error("shutdown error", "error", err)
		}
	}()

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	slog.Info("starting server", "addr", addr)

	if !devMode {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openBrowser(fmt.Sprintf("http://localhost:%d", cfg.Server.Port))
		}()
	}

	if err := app.Listen(addr, fiber.ListenConfig{
		DisableStartupMessage: true,
	}); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		slog.Debug("failed to open browser", "error", err)
	}
}

func frontendFS() fs.FS {
	sub, err := fs.Sub(frontendDist, "frontend_dist")
	if err == nil {
		_, err = fs.ReadDir(sub, ".")
		if err == nil {
			slog.Info("serving frontend from embedded filesystem")
			return sub
		}
	}
	slog.Info("serving frontend from disk", "path", "./frontend/dist")
	return os.DirFS("./frontend/dist")
}
