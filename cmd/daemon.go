package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/betom84/docker-backup/docker"
	"github.com/go-co-op/gocron"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use: "daemon",
	Run: runDaemonCmd,
}

var (
	defaultTarget   string
	defaultSchedule string
)

func init() {
	daemonCmd.Flags().StringVar(&defaultTarget, "defaultTarget", "", "Default CIFS network share address like user:pass@host/path if not defined by docker label")
	daemonCmd.Flags().StringVar(&defaultSchedule, "defaultschedule", "", "Default backup schedule if not defined by docker label")
}

func runDaemonCmd(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	logFields := logrus.Fields{}
	logFields["host"] = host

	cli, err := docker.NewClient(ctx, host)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logFields).Fatalf("failed to init docker client; %v", err)
	}
	defer cli.Close()

	containers, err := docker.FindContainerByLabel(ctx, cli, map[docker.Label]string{docker.Enabled: "true"})
	if err != nil {
		logrus.WithContext(ctx).WithFields(logFields).Fatalf("failed to find docker container by label; %v", err)
	}

	if len(containers) == 0 {
		logrus.WithContext(ctx).WithFields(logFields).Fatalf("no docker container with %s=true found", docker.Enabled)
	}

	scheduler := gocron.NewScheduler(time.Local)

	for _, c := range containers {
		logFields["container"] = c.Name

		target, err := docker.NewCifsAddress(c.Label(docker.Target, defaultTarget))
		if err != nil {
			logrus.WithContext(ctx).WithFields(logFields).Errorf("failed to parse target address; %v", err)
			continue
		}

		scheduleLabel := c.Label(docker.Schedule, defaultSchedule)

		logFields["target"] = target.String()
		logFields["schedule"] = scheduleLabel

		_, err = scheduler.Cron(scheduleLabel).Do(
			func(ctx context.Context, cli docker.Client, c *docker.Container, target docker.CifsAddress) {
				logEntry := logrus.WithContext(ctx).WithFields(logrus.Fields{
					"hostname":  cli.Hostname,
					"container": c.Name,
					"target":    target.String(),
				})

				volume, err := docker.NewCifsVolume(ctx, cli, target, fmt.Sprintf("temp_backup_target_%s", c.Name))
				if err != nil {
					logEntry.Errorf("failed to create cifs backup volume; %v", err)
					return
				}
				defer volume.Destroy(ctx)

				err = docker.Backup(ctx, cli, c, *volume)
				if err != nil {
					logEntry.Errorf("container backup failed; %v", err)
					return
				}

				logEntry.Infoln("container backup finished")
			}, ctx, cli, c, target)

		if err != nil {
			logrus.WithContext(ctx).WithFields(logFields).Errorf("failed to schedule job; %v", err)
			continue
		}

		logrus.WithContext(ctx).WithFields(logFields).Infoln("backup job scheduled")
	}

	scheduler.StartBlocking()
}
