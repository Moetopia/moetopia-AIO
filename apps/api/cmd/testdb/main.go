package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
)

var testName = regexp.MustCompile(`^moetopia_test_[a-f0-9]{16}$`)

func validate(dsn, name, action string) error {
	u, err := url.Parse(dsn)
	if err != nil || u.Scheme != "postgres" || u.Hostname() == "" || u.Path != "/moetopia_dev" || u.User == nil || u.User.Username() != "moetopia" {
		return errors.New("test provisioning requires the explicit moetopia_dev development database and moetopia role")
	}
	if !testName.MatchString(name) || (action != "create" && action != "drop") {
		return errors.New("invalid test database name or action")
	}
	return nil
}

func run() error {
	if len(os.Args) != 2 {
		return errors.New("expected create or drop")
	}
	dsn, name, action := os.Getenv("MOETOPIA_DATABASE_URL"), os.Getenv("MOETOPIA_TEST_DATABASE_NAME"), os.Args[1]
	if err := validate(dsn, name, action); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return errors.New("test provisioning connection failed; check development server and credentials")
	}
	defer conn.Close(context.Background())
	identifier := pgx.Identifier{name}.Sanitize()
	statement := "CREATE DATABASE " + identifier
	if action == "drop" {
		statement = "DROP DATABASE " + identifier + " WITH (FORCE)"
	}
	if _, err := conn.Exec(ctx, statement); err != nil {
		return fmt.Errorf("test database %s failed; check development database permissions", action)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
