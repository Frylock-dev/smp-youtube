package main

import (
	"fmt"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"os"
	"os/signal"
	"smp-youtube/config"
	api "smp-youtube/internal/api/crawler"
	"smp-youtube/internal/client/bot"
	"smp-youtube/internal/client/dropbox"
	repository "smp-youtube/internal/repository/crawler"
	service "smp-youtube/internal/service/crawler"
	"syscall"

	uc "github.com/Davincible/chromedp-undetected"
)

const (
	version = 0.1
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

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	fmt.Println(`
░██████╗███╗░░░███╗██████╗░░░░░░░██╗░░░██╗░█████╗░██╗░░░██╗████████╗██╗░░░██╗██████╗░███████╗
██╔════╝████╗░████║██╔══██╗░░░░░░╚██╗░██╔╝██╔══██╗██║░░░██║╚══██╔══╝██║░░░██║██╔══██╗██╔════╝
╚█████╗░██╔████╔██║██████╔╝█████╗░╚████╔╝░██║░░██║██║░░░██║░░░██║░░░██║░░░██║██████╦╝█████╗░░
░╚═══██╗██║╚██╔╝██║██╔═══╝░╚════╝░░╚██╔╝░░██║░░██║██║░░░██║░░░██║░░░██║░░░██║██╔══██╗██╔══╝░░
██████╔╝██║░╚═╝░██║██║░░░░░░░░░░░░░░██║░░░╚█████╔╝╚██████╔╝░░░██║░░░╚██████╔╝██████╦╝███████╗
╚═════╝░╚═╝░░░░░╚═╝╚═╝░░░░░░░░░░░░░░╚═╝░░░░╚════╝░░╚═════╝░░░░╚═╝░░░░╚═════╝░╚═════╝░╚══════╝

Started with version`, version)

	select {
	case <-ctx.Done():
		log.Errorf("ctx error: %v", ctx.Err())
		return
	case <-shutdown:
		log.Infof("received shutdown signal")
		cancel()
		return
	}
}
