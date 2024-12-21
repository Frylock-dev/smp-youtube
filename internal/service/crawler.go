package service

import (
	"context"
	"smp-youtube/internal/model"
)

type Crawler interface {
	Crawl(ctx context.Context, resources []model.Resource) error
	CrawlOnce(ctx context.Context, resources []model.Resource) (string, error)
}
