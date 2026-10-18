package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"moetopia.local/aio/api/internal/config"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence"
)

func run() error {
	lifetime := flag.Duration("expires-in", 24*time.Hour, "invitation validity, at most 720h")
	flag.Parse()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Environment != "development" {
		return errors.New("this command is restricted to the development environment")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := persistence.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	service, err := identity.New(identity.NewStore(db.Ent, nil), cfg.EncryptionKey)
	if err != nil {
		return err
	}
	code, err := service.IssueInvitation(ctx, time.Now().Add(*lifetime))
	if err != nil {
		return err
	}
	fmt.Println(code)
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Invitation creation failed; check development configuration and expiry")
		os.Exit(1)
	}
}
