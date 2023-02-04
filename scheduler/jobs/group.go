package jobs

import (
	"context"
	"fmt"

	"github.com/betom84/docker-backup/docker"
	"github.com/betom84/docker-backup/scheduler"
	"github.com/sirupsen/logrus"
)

type groupBackupJob struct {
	name       string
	ctx        context.Context
	cli        docker.Client
	containers []*docker.Container
	target     string
}

func (j *groupBackupJob) Run() {
	logEntry := logrus.WithContext(j.ctx).WithFields(logrus.Fields{
		"hostname":  j.cli.Hostname,
		"groupName": j.name,
		"target":    j.target,
	})

	volume, err := docker.NewVolume(j.ctx, j.cli, j.target, fmt.Sprintf("temp_backup_target_%s", j.name))
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

	for _, c := range j.containers {
		if c.Label(docker.Hold, "false") == "true" {
			err = c.Stop(j.ctx)
			if err != nil {
				logEntry.Errorf("failed to hold container for backup; %v", err)
				return
			}
			defer c.Start(j.ctx)
		}
	}

	for _, c := range j.containers {
		err = docker.Backup(j.ctx, j.cli, c, volume)
		if err != nil {
			logEntry.Errorf("container backup failed; %v", err)
			return
		}
	}

	logEntry.Infoln("container backup finished")
}

func NewGroupBackupJob(ctx context.Context, cli docker.Client, groupName string, group []*docker.Container, defaultTarget string) (scheduler.Job, error) {
	var target = ""

	for _, c := range group {
		if !c.HasLabel(docker.Target) {
			continue
		}

		cTarget := c.Label(docker.Target, "")
		if target != cTarget {
			logrus.WithContext(ctx).WithFields(logrus.Fields{
				"group":           groupName,
				"groupTarget":     target,
				"container":       c.Name,
				"containerTarget": cTarget,
			}).Warnf("ambiguous target address configured")
		}

		target = cTarget
	}

	if target == "" {
		if defaultTarget == "" {
			return nil, fmt.Errorf("undefined backup target")
		}

		target = defaultTarget
	}

	return &groupBackupJob{
		name:       groupName,
		ctx:        ctx,
		cli:        cli,
		containers: group,
		target:     target,
	}, nil
}
