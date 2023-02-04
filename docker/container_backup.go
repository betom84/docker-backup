package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

type BackupContainer struct {
	Container
	target Volume
	source *Container
}

func NewBackupContainer(ctx context.Context, source *Container, target Volume) (*BackupContainer, error) {
	binds := make([]string, 0, 5)
	binds = append(binds, fmt.Sprintf("%s:/target", target.GetName()))
	for _, bv := range source.Volumes() {
		binds = append(binds, fmt.Sprintf("%s:/source/%s:ro", bv, bv))
	}

	backupContainerName := fmt.Sprintf("temp_docker_backup_%s", time.Now().Format("20060102_150405"))

	c, err := NewBusyboxContainer(ctx, source.dClient, backupContainerName, binds)
	if err != nil {
		return nil, err
	}

	logrus.WithContext(ctx).WithFields(logrus.Fields{
		"container":    backupContainerName,
		"source":       source.String(),
		"targetVolume": target.String(),
	}).Debugln("backup container created")

	return &BackupContainer{Container: *c, source: source, target: target}, nil
}

func (c BackupContainer) Backup(ctx context.Context) error {
	var err error

	for _, v := range c.source.Volumes() {
		targetFolder := fmt.Sprintf("/target/%s/%s", c.dClient.Hostname, c.source.Name)
		backupFileName := fmt.Sprintf("%s_%s.tar.gz", time.Now().Format("20060102_150405"), v)

		logFields := logrus.Fields{
			"container": c.String(),
			"source":    fmt.Sprintf("%s:%s", c.source.Name, v),
			"target":    fmt.Sprintf("%s/%s/%s/%s", c.target.GetName(), c.dClient.Hostname, c.source.Name, backupFileName),
		}

		logrus.WithContext(ctx).WithFields(logFields).Debug("volume backup started")

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

		logrus.WithContext(ctx).WithFields(logFields).Info("volume backup finished")
	}

	return err
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
