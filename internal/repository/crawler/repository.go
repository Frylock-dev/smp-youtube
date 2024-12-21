package crawler

import (
	"context"
	"github.com/syndtr/goleveldb/leveldb"
	"smp-youtube/internal/repository"
)

type Repository struct {
	db *leveldb.DB
}

func NewRepository(db *leveldb.DB) repository.Crawler {
	return &Repository{db: db}
}

func (repo *Repository) SaveURLByHash(_ context.Context, hash string, url string) error {
	return repo.db.Put([]byte(hash), []byte(url), nil)
}

func (repo *Repository) HasURLByHash(_ context.Context, hash string) (bool, error) {
	return repo.db.Has([]byte(hash), nil)
}
