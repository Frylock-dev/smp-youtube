package repository

import "context"

type Crawler interface {
	SaveURLByHash(ctx context.Context, hash string, url string) error
	HasURLByHash(ctx context.Context, hash string) (bool, error)
}
