package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/betom84/docker-backup/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

type Container struct {
	Name       string
	dContainer types.Container
	dClient    Client
}

func FindContainer(ctx context.Context, client Client, name string) (*Container, error) {
	containers, err := client.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	for _, lookup := range containers {
		if !utils.Contains(lookup.Names, fmt.Sprintf("/%s", name)) {
			continue
		}

		return &Container{Name: name, dClient: client, dContainer: lookup}, nil
	}

	return nil, fmt.Errorf("container '%s' not found", name)
}

func (c Container) Start(ctx context.Context) error {
	return c.dClient.ContainerStart(ctx, c.dContainer.ID, types.ContainerStartOptions{})
}

func (c Container) Stop(ctx context.Context) error {
	return c.dClient.ContainerStop(ctx, c.dContainer.ID, nil)
}

func (c Container) Volumes() []string {
	var v []string = make([]string, 0, 3)

	for _, m := range c.dContainer.Mounts {
		if m.Type != "volume" {
			continue
		}

		v = append(v, m.Name)
	}

	return v
}

type BackupContainer struct {
	Container
	target Volume
	source *Container
}

func NewBackupContainer(ctx context.Context, source *Container, target Volume) (*BackupContainer, error) {
	BackupContainerImageName := "busybox"

	out, err := source.dClient.ImagePull(ctx, BackupContainerImageName, types.ImagePullOptions{})
	if err != nil {
		return nil, err
	}
	defer out.Close()

	binds := make([]string, 0, 5)
	binds = append(binds, fmt.Sprintf("%s:/target", target.Name))
	for _, bv := range source.Volumes() {
		binds = append(binds, fmt.Sprintf("%s:/source/%s:ro", bv, bv))
	}

	backupContainerName := fmt.Sprintf("temp_docker_backup_%s", time.Now().Format("20060102_150405"))
	cmd := []string{"/bin/sh", "-c", "tail -f /dev/null"}

	_, err = source.dClient.ContainerCreate(ctx, &container.Config{Image: BackupContainerImageName, Cmd: cmd}, &container.HostConfig{Binds: binds}, nil, nil, backupContainerName)
	if err != nil {
		return nil, err
	}

	c, err := FindContainer(ctx, source.dClient, backupContainerName)
	if err != nil {
		return nil, err
	}

	return &BackupContainer{Container: *c, source: source, target: target}, nil
}

func (c *BackupContainer) Destroy(ctx context.Context) error {
	err := c.Stop(ctx)
	if err != nil {
		return err
	}

	return c.dClient.ContainerRemove(ctx, c.dContainer.ID, types.ContainerRemoveOptions{})
}

func (c BackupContainer) Backup(ctx context.Context) error {
	var err error

	for _, v := range c.source.Volumes() {
		targetFolder := fmt.Sprintf("/target/%s/%s", c.dClient.Host, c.source.Name)
		backupFileName := fmt.Sprintf("%s_%s.tar.gz", time.Now().Format("20060102_150405"), v)

		fmt.Printf("%s:%s >>> %s/%s/%s/%s\n", c.source.Name, v, c.target.Resource, c.dClient.Host, c.source.Name, backupFileName)

		err = c.exec(ctx, []string{"mkdir", "-p", targetFolder})
		if err != nil {
			fmt.Println(err)
		}

		cmd := []string{
			"/bin/tar",
			"czvf",
			fmt.Sprintf("%s/%s", targetFolder, backupFileName),
			fmt.Sprintf("--directory=/source/%s", v),
			".",
		}
		err = c.exec(ctx, cmd)
		if err != nil {
			fmt.Println(err)
		}
	}

	return err
}

func (c BackupContainer) exec(ctx context.Context, cmd []string) error {
	execConfig := types.ExecConfig{AttachStdout: true, AttachStderr: true, Cmd: cmd}
	execResp, err := c.dClient.ContainerExecCreate(ctx, c.dContainer.ID, execConfig)
	if err != nil {
		return err
	}

	att, err := c.dClient.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	if err != nil {
		return err
	}

	defer att.Close()
	io.Copy(os.Stdout, att.Reader)

	return c.dClient.ContainerExecStart(ctx, execResp.ID, types.ExecStartCheck{})
}
