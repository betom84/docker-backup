package docker

import (
	"context"

	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

type Client struct {
	*client.Client
	Hostname string
}

var DefaultDockerHost = client.DefaultDockerHost

func NewClient(ctx context.Context, host string) (Client, error) {
	cli, err := client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
	if err != nil {
		return Client{}, err
	}

	i, err := cli.Info(ctx)
	if err != nil {
		cli.Close()
		return Client{}, err
	}

	logrus.WithField("hostname", i.Name).Debugln("docker client connection established")

	return Client{cli, i.Name}, nil
}

func (c *Client) Close() {
	logrus.WithField("hostname", c.Hostname).Debugln("docker client connection closed")
	c.Client.Close()
}
