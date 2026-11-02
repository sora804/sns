// Package config は環境変数から設定を読む。
package config

import (
	"errors"
	"os"
)

type Config struct {
	Addr         string
	DatabaseURL  string
	CookieSecure bool
}

func Load() (*Config, error) {
	c := &Config{
		Addr:        ":" + getenv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		// 本番は HTTPS 前提なので Secure を既定で有効にする。ローカルの http で試すときだけ false にする。
		CookieSecure: getenv("COOKIE_SECURE", "true") != "false",
	}
	if c.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL が設定されていません")
	}
	return c, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
