package cmd

import (
	"context"
	"fmt"

	"github.com/betom84/docker-backup/docker"
	"github.com/betom84/docker-backup/scheduler"
	"github.com/betom84/docker-backup/scheduler/jobs"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	defaultTarget   string
	defaultSchedule string
)

func NewDaemonCommand() *cobra.Command {
	daemonCmd := &cobra.Command{
		Use:  "daemon",
		RunE: runDaemonCmd,
	}

	daemonCmd.Flags().StringVar(&defaultTarget, "defaultTarget", "", "Default CIFS network share address like user:pass@host/path if not defined by docker label")
	daemonCmd.Flags().StringVar(&defaultSchedule, "defaultSchedule", "", "Default backup schedule if not defined by docker label")

	return daemonCmd
}

func runDaemonCmd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	cli, err := docker.NewClient(ctx, host)
	if err != nil {
		return fmt.Errorf("failed to init docker client; %v", err)
	}
	defer cli.Close()

	containerGroups, err := docker.FindContainerGroupsByLabel(ctx, cli, map[docker.Label]string{docker.Enabled: "true"})
	if err != nil {
		return fmt.Errorf("failed to find docker container by label; %w", err)
	}

	if len(containerGroups) == 0 {
		return fmt.Errorf("no suitable docker container found")
	}

	s := scheduler.NewScheduler()

	for name, group := range containerGroups {
		if name == "" {
			scheduleSingleBackupJobs(ctx, cli, s, group, defaultTarget)
		} else {
			scheduleGroupBackupJob(ctx, cli, s, name, group, defaultTarget)
		}
	}

	s.Run(ctx)

	return nil
}

func scheduleSingleBackupJobs(ctx context.Context, cli docker.Client, s *scheduler.Scheduler, containers []*docker.Container, defaultTarget string) {

	logFields := logrus.Fields{}
	logFields["hostname"] = cli.Hostname

	for _, c := range containers {
		schedule := c.Label(docker.Schedule, defaultSchedule)

		logFields["container"] = c.Name
		logFields["schedule"] = schedule

		j, err := jobs.NewSingleBackupJob(ctx, cli, c, defaultTarget)
		if err == nil {
			err = s.Add(schedule, j)
		}

		if err != nil {
			logrus.WithContext(ctx).WithFields(logFields).Errorf("failed to schedule single backup job; %v", err)
		} else {
			logrus.WithContext(ctx).WithFields(logFields).Infoln("single backup job scheduled")
		}
	}
}

func scheduleGroupBackupJob(ctx context.Context, cli docker.Client, s *scheduler.Scheduler, groupName string, containers []*docker.Container, defaultTarget string) {

	logFields := logrus.Fields{}
	logFields["host"] = host
	logFields["name"] = groupName

	schedule := ""
	for _, c := range containers {
		cSchedule := c.Label(docker.Schedule, schedule)
		if schedule != "" && schedule != cSchedule {
			logrus.WithContext(ctx).WithFields(logFields).Warnf("ambiguous group schedule configured")
		}

		schedule = cSchedule
	}

	if schedule == "" {
		schedule = defaultSchedule
	}

	logFields["schedule"] = schedule

	j, err := jobs.NewGroupBackupJob(ctx, cli, groupName, containers, defaultTarget)
	if err == nil {
		err = s.Add(schedule, j)
	}

	if err != nil {
		logrus.WithContext(ctx).WithFields(logFields).Errorf("failed to schedule group backup job; %v", err)
	} else {
		logrus.WithContext(ctx).WithFields(logFields).Infoln("group backup job scheduled")
	}
}
