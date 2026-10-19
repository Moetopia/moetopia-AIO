package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/valkey-io/valkey-go"
	"moetopia.local/aio/api/internal/administration"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/config"
	"moetopia.local/aio/api/internal/cryptobox"
	"moetopia.local/aio/api/internal/developer"
	"moetopia.local/aio/api/internal/httpserver"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/relyingparty"
	"moetopia.local/aio/api/internal/sessions"
)

func main() {
	if err := run(); err != nil {
		slog.Error("API stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	box, err := cryptobox.New(cfg.EncryptionKey)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startup, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	db, err := persistence.Open(startup, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Ready(startup); err != nil {
		return err
	}
	cache, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{cfg.ValkeyAddress}, Password: cfg.ValkeyPassword, DisableCache: true, Dialer: net.Dialer{Timeout: 5 * time.Second}})
	if err != nil {
		return errors.New("Valkey connection failed; check MOETOPIA_VALKEY_ADDRESS")
	}
	defer cache.Close()
	key, err := authorization.LoadActiveKey(startup, db, box)
	if err != nil {
		return err
	}
	if cfg.Environment == "production" {
		return errors.New("production account email transport is not implemented")
	}
	if cfg.SMTPAddress == "" || cfg.SMTPFrom == "" {
		return errors.New("MOETOPIA_SMTP_ADDRESS and MOETOPIA_SMTP_FROM are required")
	}
	outbox := notifications.New(db.Ent, box, notifications.DevelopmentSMTP{Address: cfg.SMTPAddress, From: cfg.SMTPFrom})
	webSessions, err := sessions.New(db.Ent, cfg.EncryptionKey, cfg.AccountOrigin, cfg.Environment == "production")
	if err != nil {
		return err
	}
	accounts, err := identity.New(identity.NewStore(db.Ent, outbox).WithSessions(webSessions), cfg.EncryptionKey)
	if err != nil {
		return err
	}
	provider, err := authorization.NewProvider(authorization.NewStore(db.Ent, box), key, cfg.Issuer, cfg.EncryptionKey, accounts)
	if err != nil {
		return err
	}
	workerDone := make(chan struct{})
	protocol := http.NewServeMux()
	protocol.HandleFunc("POST /oauth/token", provider.Token)
	protocol.HandleFunc("GET /oauth/jwks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(provider.JWKS())
	})
	portal, err := relyingparty.New(db.Ent, box, accounts, cfg.Issuer, cfg.EncryptionKey, cfg.Environment == "production", protocol)
	if err != nil {
		return err
	}
	admin := httpserver.AdminEndpoints{}
	if cfg.AdminOrigin != "" {
		adminWeb, err := sessions.NewAdmin(db.Ent, cfg.EncryptionKey, cfg.AdminOrigin, strings.HasPrefix(cfg.AdminOrigin, "https://"))
		if err != nil {
			return err
		}
		admin.Service = administration.New(db.Ent, box, accounts, outbox, adminWeb)
		admin.RP, err = relyingparty.NewAdmin(db.Ent, box, admin.Service, cfg.Issuer, cfg.AdminOrigin, cfg.EncryptionKey, strings.HasPrefix(cfg.AdminOrigin, "https://"), protocol)
		if err != nil {
			return err
		}
	}
	go func() {
		defer close(workerDone)
		runWorkers(ctx, outbox, accounts, webSessions, provider, portal, admin.Service)
	}()
	defer func() { stop(); <-workerDone }()
	handler := httpserver.New(httpserver.Dependencies{
		RateNamespace: cfg.RateNamespace(), DatabaseReady: db.Ready,
		CacheReady: func(ctx context.Context) error { return cache.Do(ctx, cache.B().Ping().Build()).Error() },
		Tokens:     provider, Valkey: cache, Portal: httpserver.PortalEndpoints{Service: portal},
		Developers: developer.New(db.Ent, provider, accounts, cfg.Issuer),
		Admin:      admin,
		Accounts:   httpserver.AccountEndpoints{Identity: accounts, Sessions: webSessions, Cache: cache, Authorization: provider},
	})
	server := &http.Server{Addr: cfg.Listen, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	errorsCh := make(chan error, 1)
	go func() { errorsCh <- server.ListenAndServe() }()
	slog.Info("API listening", "address", cfg.Listen)
	select {
	case err := <-errorsCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
	return nil
}
