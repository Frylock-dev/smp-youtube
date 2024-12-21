package crawler

import (
	"context"
	"encoding/json"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"smp-youtube/internal/client/bot"
	"smp-youtube/internal/model"
	"smp-youtube/internal/service"
)

const (
	queue = "youtube-browser"
)

type API struct {
	crawlerService service.Crawler
	botClient      *bot.Client
}

func NewAPI(
	ctx context.Context,
	nc *nats.Conn,
	crawler service.Crawler,
	client *bot.Client,
) error {
	api := &API{
		crawlerService: crawler,
		botClient:      client,
	}

	_, err := nc.QueueSubscribe("youtube-crawler", queue, api.Crawl(ctx))
	if err != nil {
		return err
	}

	_, err = nc.QueueSubscribe("youtube-crawler-once", queue, api.CrawlOnce(ctx))
	if err != nil {
		return err
	}

	return nil
}

func (api *API) Crawl(ctx context.Context) nats.MsgHandler {
	return func(msg *nats.Msg) {
		var resources []model.Resource

		log.WithFields(log.Fields{
			"data": string(msg.Data),
		}).Info("received message")

		err := json.Unmarshal(msg.Data, &resources)
		if err != nil {
			log.Error(err)
		}

		err = api.crawlerService.Crawl(ctx, resources)
		if err != nil {
			log.Error(err)
		}
	}
}

func (api *API) CrawlOnce(ctx context.Context) nats.MsgHandler {
	return func(msg *nats.Msg) {
		var resources []model.Resource

		log.WithFields(log.Fields{
			"data": string(msg.Data),
		}).Info("received message")

		err := json.Unmarshal(msg.Data, &resources)
		if err != nil {
			log.Error(err)
		}

		link, err := api.crawlerService.CrawlOnce(ctx, resources)
		if err != nil {
			log.Error(err)
		}

		err = api.botClient.SendLink(resources[0].ID, link)
		if err != nil {
			log.Error(err)
		}
	}
}
