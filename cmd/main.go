package main

import (
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"smp-youtube/config"
	api "smp-youtube/internal/api/crawler"
	"smp-youtube/internal/client/bot"
	"smp-youtube/internal/client/dropbox"
	repository "smp-youtube/internal/repository/crawler"
	service "smp-youtube/internal/service/crawler"

	uc "github.com/Davincible/chromedp-undetected"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		return
	}

	nc, err := nats.Connect(cfg.NatsDSN)
	if err != nil {
		log.Fatal(err)
	}

	defer nc.Close()

	ctx, cancel, err := uc.New(uc.NewConfig(
		uc.WithNoSandbox(true),
		uc.WithUserDataDir(cfg.BrowserUserData),
	))
	if err != nil {
		return
	}

	defer cancel()

	db, err := leveldb.OpenFile(cfg.DatabaseDSN, nil)
	if err != nil {
		return
	}

	repo := repository.NewRepository(db)

	botClient := bot.NewClient(nc)
	dropboxClient := dropbox.NewClient(
		cfg.DropBoxRefreshToken,
		cfg.DropBoxAppKey,
		cfg.DropBoxAppSecret,
	)

	crawler := service.NewService(
		cfg.CookiesPath,
		cfg.TmpOutPutPath,
		dropboxClient,
		repo,
	)

	err = api.NewAPI(ctx, nc, crawler, botClient)
	if err != nil {
		log.Fatal(err)
	}

	select {
	case <-ctx.Done():
		log.Error(ctx.Err())
		return
	}
}
