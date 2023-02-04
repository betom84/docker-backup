package helper

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

type BusyboxContainer struct {
	Host       string
	Name       string
	ID         string
	VolumeName string
}

func NewBusyboxContainer(t *testing.T, name string, labels map[string]string) BusyboxContainer {
	t.Helper()

	busybox := BusyboxContainer{
		Name:       name,
		VolumeName: fmt.Sprintf("busybox_volume_%s", name),
	}

	cli, err := client.NewClientWithOpts(client.WithHost(client.DefaultDockerHost), client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cli.Close() })

	i, err := cli.Info(context.TODO())
	if err == nil {
		busybox.Host = i.Name
	}

	out, err := cli.ImagePull(context.TODO(), "busybox", types.ImagePullOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	io.Copy(os.Stdout, out)

	_, err = cli.VolumeCreate(context.TODO(), volume.VolumeCreateBody{Name: busybox.VolumeName})
	if err != nil {
		t.Fatal(err)
	}

	c, err := cli.ContainerCreate(context.TODO(), &container.Config{
		Image:  "busybox",
		Cmd:    []string{"/bin/sh", "-c", "tail -f /dev/null"},
		Labels: labels,
	}, &container.HostConfig{
		Binds:      []string{fmt.Sprintf("%s:/data", busybox.VolumeName)},
		Privileged: true,
	}, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"docker-backup-test": &network.EndpointSettings{NetworkID: GetOrCreateNetwork(t, cli)},
		},
	}, nil, name)

	if err != nil {
		t.Fatal(err)
	}

	busybox.ID = c.ID

	t.Cleanup(func() {
		cli.ContainerRemove(context.TODO(), c.ID, types.ContainerRemoveOptions{RemoveVolumes: true, Force: true})
	})

	err = cli.ContainerStart(context.TODO(), c.ID, types.ContainerStartOptions{})
	if err != nil {
		t.Fatal(err)
	}

	return busybox
}
