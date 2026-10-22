package config

import (
	"errors"
	"io"
	"os"
	"strings"
)

var secretKeys = []string{"MOETOPIA_DATABASE_URL", "MOETOPIA_VALKEY_PASSWORD", "MOETOPIA_ENCRYPTION_KEY", "MOETOPIA_SMTP_PASSWORD", "MOETOPIA_GITHUB_CLIENT_SECRET", "MOETOPIA_GOOGLE_CLIENT_SECRET"}

func loadSecrets(get func(string) string) (map[string]string, error) {
	values := map[string]string{}
	for _, key := range secretKeys {
		path := get(key + "_FILE")
		if path == "" {
			continue
		}
		if get(key) != "" {
			return nil, errors.New(key + " and its _FILE setting cannot both be set")
		}
		file, err := os.Open(path)
		if err != nil {
			return nil, errors.New("cannot read " + key + "_FILE")
		}
		data, err := io.ReadAll(io.LimitReader(file, 16385))
		_ = file.Close()
		if err != nil || len(data) > 16384 {
			return nil, errors.New("invalid " + key + "_FILE")
		}
		values[key] = strings.TrimRight(string(data), "\r\n")
	}
	return values, nil
}
