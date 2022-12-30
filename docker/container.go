package docker

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/betom84/docker-backup/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/sirupsen/logrus"
)

type Label string

var (
	Enabled  Label = "de.betom.docker-backup.enabled"
	Schedule Label = "de.betom.docker-backup.schedule"
	Target   Label = "de.betom.docker-backup.target"
)

type Container struct {
	Name       string
	dContainer types.Container
	dClient    Client
}

func FindContainerByLabel(ctx context.Context, client Client, labels map[Label]string) ([]*Container, error) {
	result := make([]*Container, 0, 5)

	f := make([]filters.KeyValuePair, 0)
	for k, v := range labels {
		f = append(f, filters.Arg("label", fmt.Sprintf("%s=%s", k, v)))
	}

	containers, err := client.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(f...),
	})

	if err != nil {
		return result, err
	}

	for _, lookup := range containers {
		name := lookup.ID
		if len(lookup.Names) > 0 {
			name = lookup.Names[0]
		}

		result = append(result, &Container{Name: name, dClient: client, dContainer: lookup})
	}

	return result, nil
}

func FindContainerByName(ctx context.Context, client Client, name string) (*Container, error) {
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

func Backup(ctx context.Context, cli Client, source *Container, target Volume) error {
	backup, err := NewBackupContainer(ctx, source, target)
	if err != nil {
		return err
	}
	defer backup.Destroy(ctx)

	err = backup.Start(ctx)
	if err != nil {
		return err
	}

	err = backup.Backup(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (c Container) Start(ctx context.Context) error {
	err := c.dClient.ContainerStart(ctx, c.dContainer.ID, types.ContainerStartOptions{})

	logrus.WithContext(ctx).WithFields(logrus.Fields{
		"container": c.String(),
		"error":     err,
	}).Debug("container started")

	return err
}

func (c Container) Stop(ctx context.Context) error {
	err := c.dClient.ContainerStop(ctx, c.dContainer.ID, nil)

	logrus.WithContext(ctx).WithFields(logrus.Fields{
		"container": c.String(),
		"error":     err,
	}).Debug("container stopped")

	return err
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

func (c Container) Label(key Label, fallback string) string {
	if v, ok := c.dContainer.Labels[string(key)]; ok {
		return v
	}

	logrus.New().WithField("container", c.Name).Warningf("label '%s' not find, using fallback value '%s'", key, fallback)
	return fallback
}

func (c Container) String() string {
	return c.Name
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

	c, err := FindContainerByName(ctx, source.dClient, backupContainerName)
	if err != nil {
		return nil, err
	}

	logrus.WithContext(ctx).WithFields(logrus.Fields{
		"container": backupContainerName,
		"source":    source.String(),
		"target":    target.String(),
	}).Debugln("backup container created")

	return &BackupContainer{Container: *c, source: source, target: target}, nil
}

func (c *BackupContainer) Destroy(ctx context.Context) error {
	err := c.Stop(ctx)
	if err != nil {
		return err
	}

	err = c.dClient.ContainerRemove(ctx, c.dContainer.ID, types.ContainerRemoveOptions{})

	logrus.WithContext(ctx).WithFields(logrus.Fields{
		"container": c.String(),
		"error":     err,
	}).Debugln("container destroyed")

	return err
}

func (c BackupContainer) Backup(ctx context.Context) error {
	var err error

	for _, v := range c.source.Volumes() {
		targetFolder := fmt.Sprintf("/target/%s/%s", c.dClient.Hostname, c.source.Name)
		backupFileName := fmt.Sprintf("%s_%s.tar.gz", time.Now().Format("20060102_150405"), v)

		logFields := logrus.Fields{
			"container": c.String,
			"source":    fmt.Sprintf("%s:%s", c.source.Name, v),
			"target":    fmt.Sprintf("%s/%s/%s/%s", c.target.Resource, c.dClient.Hostname, c.source.Name, backupFileName),
		}

		logrus.WithContext(ctx).WithFields(logFields).Debug("container volume backup started")

		cmd := []string{"mkdir", "-p", targetFolder}
		err = c.exec(ctx, cmd)
		if err != nil {
			logrus.WithContext(ctx).WithFields(logFields).WithField("cmd", cmd).Errorf("failed to create backup target folder; %v", err)
			continue
		}

		cmd = []string{
			"/bin/tar",
			"czvf",
			fmt.Sprintf("%s/%s", targetFolder, backupFileName),
			fmt.Sprintf("--directory=/source/%s", v),
			".",
		}

		err = c.exec(ctx, cmd)
		if err != nil {
			logrus.WithContext(ctx).WithFields(logFields).WithField("cmd", cmd).Errorf("failed to exec backup command; %v", err)
			continue
		}

		logrus.WithContext(ctx).WithFields(logFields).Info("container volume backup finished")
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

	if logrus.GetLevel() == logrus.TraceLevel {
		io.Copy(logrus.StandardLogger().Out, att.Reader)
	}

	return c.dClient.ContainerExecStart(ctx, execResp.ID, types.ExecStartCheck{})
}
