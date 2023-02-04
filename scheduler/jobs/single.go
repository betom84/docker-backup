package jobs

import (
	"context"
	"fmt"

	"github.com/betom84/docker-backup/docker"
	"github.com/betom84/docker-backup/scheduler"
	"github.com/sirupsen/logrus"
)

type singleBackupJob struct {
	ctx       context.Context
	cli       docker.Client
	container *docker.Container
	target    string
}

func (j *singleBackupJob) Run() {
	logEntry := logrus.WithContext(j.ctx).WithFields(logrus.Fields{
		"hostname":  j.cli.Hostname,
		"container": j.container.Name,
		"target":    j.target,
	})

	volume, err := docker.NewVolume(j.ctx, j.cli, j.target, fmt.Sprintf("temp_backup_target_%s", j.container.Name))
	if err != nil {
		logEntry.Errorf("failed to init backup volume; %v", err)
		return
	}

	err = volume.Create(j.ctx)
	if err != nil {
		logEntry.Errorf("failed to create backup volume; %v", err)
		return
	}
	defer volume.Destroy(j.ctx)

	if j.container.Label(docker.Hold, "false") == "true" {
		err = j.container.Stop(j.ctx)
		if err != nil {
			logEntry.Errorf("failed to hold container for backup; %v", err)
			return
		}
		defer j.container.Start(j.ctx)
	}

	err = docker.Backup(j.ctx, j.cli, j.container, volume)
	if err != nil {
		logEntry.Errorf("container backup failed; %v", err)
		return
	}

	logEntry.Infoln("container backup finished")
}

func NewSingleBackupJob(ctx context.Context, cli docker.Client, c *docker.Container, defaultTarget string) (scheduler.Job, error) {

	if !c.HasLabel(docker.Target) && defaultTarget == "" {
		return nil, fmt.Errorf("undefined backup target")
	}

	return &singleBackupJob{
		ctx:       ctx,
		cli:       cli,
		container: c,
		target:    c.Label(docker.Target, defaultTarget),
	}, nil
}
