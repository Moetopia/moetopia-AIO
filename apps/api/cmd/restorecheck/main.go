package main

import (
	"context"
	"flag"
	"log/slog"
	"moetopia.local/aio/api/internal/config"
	"moetopia.local/aio/api/internal/restoration"
	"os"
	"time"
)

func main() {
	invalidate := flag.Bool("invalidate-transient", false, "invalidate restored sessions, challenges, protocol credentials and queued mail")
	flag.Parse()
	cfg, err := config.Load()
	if err == nil && flag.NArg() != 0 {
		err = os.ErrInvalid
	}
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		err = restoration.Check(ctx, cfg.DatabaseURL, cfg.EncryptionKey, *invalidate)
	}
	if err != nil {
		slog.Error("restore check failed; target, schema or original encryption key is invalid; no credentials were displayed")
		os.Exit(1)
	}
	slog.Info("isolated restored schema and signing material verified", "transient_credentials_invalidated", *invalidate)
}
