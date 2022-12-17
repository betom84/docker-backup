package docker

import (
	"fmt"

	"github.com/docker/docker/client"
)

type Client struct {
	*client.Client
	Host string
}

func NewClient(tcpHost string) (Client, error) {
	cli, err := client.NewClientWithOpts(client.WithHost(fmt.Sprintf("tcp://%s:2375", tcpHost)), client.WithAPIVersionNegotiation())
	if err != nil {
		return Client{}, err
	}

	return Client{cli, tcpHost}, nil
}
