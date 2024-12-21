package bot

import (
	"encoding/json"
	"github.com/nats-io/nats.go"
	"smp-youtube/internal/model"
)

const (
	subj = "bot"
)

type Client struct {
	nc *nats.Conn
}

func NewClient(nc *nats.Conn) *Client {
	return &Client{
		nc: nc,
	}
}

func (c *Client) SendLink(id int, link string) error {
	marshal, err := json.Marshal(&model.Resource{
		ID:  id,
		URL: link,
	})
	if err != nil {
		return err
	}

	err = c.nc.Publish(subj, marshal)
	if err != nil {
		return err
	}

	return nil
}
