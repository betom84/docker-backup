package docker

import (
	"context"
	"fmt"
	"io"
	"strings"

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
	Hold     Label = "de.betom.docker-backup.hold"
	Group    Label = "de.betom.docker-backup.group"
)

type Container struct {
	Name       string
	dContainer types.Container
	dClient    Client
}

func NewBusyboxContainer(ctx context.Context, dClient Client, name string, binds []string) (*Container, error) {
	imageName := "busybox"
	cmd := []string{"/bin/sh", "-c", "tail -f /dev/null"}

	out, err := dClient.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return nil, err
	}
	defer out.Close()

	_, err = dClient.ContainerCreate(ctx, &container.Config{Image: imageName, Cmd: cmd}, &container.HostConfig{Binds: binds}, nil, nil, name)
	if err != nil {
		return nil, err
	}

	return FindContainerByName(ctx, dClient, name)
}

func FindContainerGroupsByLabel(ctx context.Context, client Client, labels map[Label]string) (map[string][]*Container, error) {
	result := make(map[string][]*Container, 0)

	containers, err := FindContainerByLabel(ctx, client, labels)
	if err != nil {
		return result, err
	}

	for _, c := range containers {
		if l, ok := result[c.dContainer.Labels[string(Group)]]; ok {
			result[c.dContainer.Labels[string(Group)]] = append(l, c)
		} else {
			result[c.dContainer.Labels[string(Group)]] = []*Container{c}
		}
	}

	return result, nil
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
			name = strings.TrimPrefix(lookup.Names[0], "/")
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

func (c Container) Destroy(ctx context.Context) error {
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

func (c Container) exec(ctx context.Context, cmd []string) error {
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

func (c Container) Label(key Label, fallback string) string {
	if v, ok := c.dContainer.Labels[string(key)]; ok && v != "" {
		return v
	}

	logrus.New().WithField("container", c.Name).Warningf("label '%s' not found or empty, using fallback value '%s'", key, fallback)
	return fallback
}

func (c Container) HasLabel(key Label) bool {
	_, ok := c.dContainer.Labels[string(key)]
	return ok
}

func (c Container) String() string {
	return c.Name
}
