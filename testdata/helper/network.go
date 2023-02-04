package helper

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

var networkID = ""

func GetOrCreateNetwork(t *testing.T, cli *client.Client) string {
	if networkID == "" {
		n, err := cli.NetworkCreate(context.TODO(), "docker-backup-test", types.NetworkCreate{})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			cli.NetworkRemove(context.TODO(), n.ID)
			networkID = ""
		})

		networkID = n.ID
	}

	return networkID
}
