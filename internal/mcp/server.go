// Package mcp exposes the MECS application to AI agents via the Model Context
// Protocol (MCP), served over streamable HTTP on a configurable localhost
// address. All tools operate on the same store and config used by the Wails
// frontend, so AI actions are immediately visible in the UI and vice versa.
package mcp

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/raditzlawliet/test-mass-email/internal/config"
	"github.com/raditzlawliet/test-mass-email/internal/store"
)

// Server embeds an MCP server into the MECS process. Tool state lives in the
// shared store; the current global config is always read through cfgProvider so
// SaveConfig reloads are picked up immediately.
type Server struct {
	store       *store.Store
	cfgProvider func() *config.Config
	configPath  string
	version     string
	reload      func() error

	mcp        *mcp.Server
	httpServer *http.Server
	startedCfg config.MCPConfig
	actualAddr string
	lastErr    string
	mu         sync.Mutex

	// lifecycleBusy coalesces overlapping reconciliation/restart work so a
	// second listener is never started by concurrent Reconcile calls.
	lifecycleBusy bool
}

// NewServer creates the MCP server and registers all tools.
// reload is invoked after a successful config save (e.g. from set_config).
func NewServer(st *store.Store, cfgProvider func() *config.Config, configPath, version string, reload func() error) *Server {
	s := &Server{
		store:       st,
		cfgProvider: cfgProvider,
		configPath:  configPath,
		version:     version,
		reload:      reload,
	}
	s.mcp = mcp.NewServer(&mcp.Implementation{Name: "mecs", Version: version}, nil)
	s.RegisterTools(s.mcp)
	return s
}

// Running reports whether the HTTP listener is currently active.
func (s *Server) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.httpServer != nil
}

// Reconcile starts, stops, or restarts the HTTP listener to match the current
// mcp config section. Safe to call on every config change. Concurrent calls
// coalesce: while a reconciliation (or async restart) is in progress, later
// calls are dropped - the running one already matches the latest config it saw.
func (s *Server) Reconcile() {
	cfg := s.cfgProvider()

	s.mu.Lock()
	if s.lifecycleBusy {
		s.mu.Unlock()
		return
	}
	s.lifecycleBusy = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.lifecycleBusy = false
		s.mu.Unlock()
	}()

	if !cfg.MCP.Enabled {
		s.stopAsync()
		return
	}
	if !s.Running() {
		if err := s.Start(); err != nil {
			slog.Error("failed to start MCP server", "error", err)
		}
		return
	}

	s.mu.Lock()
	changed := s.startedCfg.Host != cfg.MCP.Host ||
		s.startedCfg.Port != cfg.MCP.Port ||
		s.startedCfg.Token != cfg.MCP.Token
	s.mu.Unlock()
	if changed {
		s.restartAsync()
	}
}

// Start binds the streamable HTTP listener on the configured host:port and
// serves the MCP endpoint at /mcp (plus a plain /mcp/health endpoint).
func (s *Server) Start() error {
	cfg := s.cfgProvider()
	if !cfg.MCP.Enabled {
		return nil
	}
	addr := net.JoinHostPort(cfg.MCP.Host, strconv.Itoa(cfg.MCP.Port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		s.mu.Lock()
		s.lastErr = err.Error()
		s.mu.Unlock()
		return fmt.Errorf("mcp listen %s: %w", addr, err)
	}

	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s.mcp
	}, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if !isLocalOrigin(r) {
			http.Error(w, "forbidden: non-local origin", http.StatusForbidden)
			return
		}
		if cfg.MCP.Token != "" && !authorized(r, cfg.MCP.Token) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		mcpHandler.ServeHTTP(w, r)
	})
	mux.HandleFunc("/mcp/health", func(w http.ResponseWriter, r *http.Request) {
		if !isLocalOrigin(r) {
			http.Error(w, "forbidden: non-local origin", http.StatusForbidden)
			return
		}
		if cfg.MCP.Token != "" && !authorized(r, cfg.MCP.Token) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","campaign_state":%q}`, s.store.GetState())
	})

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	actualAddr := ln.Addr().String()
	s.mu.Lock()
	s.httpServer = srv
	s.startedCfg = cfg.MCP
	s.actualAddr = actualAddr
	s.lastErr = ""
	s.mu.Unlock()

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			slog.Warn("MCP server stopped unexpectedly", "error", err)
		}
	}()

	slog.Info("MCP server started", "addr", actualAddr)
	return nil
}

// ListenAddr returns the bound address of the active listener, or empty when
// the server is not running. Exposed for tests and health reporting.
func (s *Server) ListenAddr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.actualAddr
}

// Status reports the live MCP server state: running (listener active, with the
// bound address), starting (enabled but not yet bound), error (last start
// failed, with the message), or stopped (disabled).
func (s *Server) Status() (status, addr, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.httpServer != nil {
		return "running", s.actualAddr, ""
	}
	if s.lastErr != "" {
		return "error", "", s.lastErr
	}
	if s.cfgProvider().MCP.Enabled {
		return "starting", "", ""
	}
	return "stopped", "", ""
}

// Stop gracefully shuts down the HTTP listener, waiting for in-flight requests.
func (s *Server) Stop() {
	s.mu.Lock()
	srv := s.httpServer
	s.httpServer = nil
	s.lastErr = ""
	s.mu.Unlock()
	if srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Warn("MCP server shutdown", "error", err)
	}
	slog.Info("MCP server stopped")
}

// stopAsync stops the listener without blocking the caller. Used when stopping
// from inside a request handler (Shutdown waits for in-flight requests, which
// would otherwise deadlock).
func (s *Server) stopAsync() {
	if !s.Running() {
		return
	}
	go s.Stop()
}

// restartAsync restarts the listener with current settings. Also used from
// inside request handlers, hence async. Overlapping calls coalesce via the
// lifecycleBusy guard.
func (s *Server) restartAsync() {
	if !s.Running() {
		return
	}
	go func() {
		s.mu.Lock()
		if s.lifecycleBusy {
			s.mu.Unlock()
			return
		}
		s.lifecycleBusy = true
		s.mu.Unlock()
		defer func() {
			s.mu.Lock()
			s.lifecycleBusy = false
			s.mu.Unlock()
		}()
		s.Stop()
		if err := s.Start(); err != nil {
			slog.Error("failed to restart MCP server", "error", err)
		}
	}()
}

func authorized(r *http.Request, token string) bool {
	h := r.Header.Get("Authorization")
	want := "Bearer " + token
	return len(h) == len(want) && subtle.ConstantTimeCompare([]byte(h), []byte(want)) == 1
}

// isLocalOrigin rejects browser-origin requests that do not originate from the
// local machine (defense against malicious web pages targeting the localhost
// endpoint). MCP clients that are not browsers send no Origin header.
func isLocalOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// boolPtr returns a pointer to v for *bool annotation fields.
func boolPtr(v bool) *bool { return &v }
