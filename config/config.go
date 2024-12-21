package config

import (
	"github.com/caarlos0/env/v10"
	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	BrowserUserData     string `env:"BROWSER_USER_DATA" envDefault:""`
	DropBoxAppKey       string `env:"DROPBOX_APP_KEY" envDefault:""`
	DropBoxAppSecret    string `env:"DROPBOX_APP_SECRET" envDefault:""`
	DropBoxRefreshToken string `env:"DROPBOX_REFRESH_TOKEN" envDefault:""`
	NatsDSN             string `env:"NATS_DSN" envDefault:"nats://localhost:4222"`
	CookiesPath         string `env:"COOKIES_PATH" envDefault:"./resources/cookies/cookie.txt"`
	TmpOutPutPath       string `env:"TMP_OUTPUT_PATH" envDefault:"./tmp/videos"`
	DatabaseDSN         string `env:"DATABASE_DSN" envDefault:"./tmp/db"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}

	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
