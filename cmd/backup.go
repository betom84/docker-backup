package cmd

import (
	"strings"

	"github.com/betom84/docker-backup/docker"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup Docker container volumes to CIFS network share",
	Run:   runBackupCmd,
}

var (
	container string
	target    string
	hold      bool
)

func init() {
	backupCmd.Flags().StringVar(&container, "container", "", "Comma separated list of Docker container names (required)")
	backupCmd.Flags().StringVar(&target, "target", "", "CIFS network share address like user:pass@host/path (required)")
	backupCmd.Flags().BoolVar(&hold, "hold", false, "Hold container during backup")

	backupCmd.MarkFlagRequired("container")
	backupCmd.MarkFlagRequired("target")
}

func runBackupCmd(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	cli, err := docker.NewClient(ctx, host)
	if err != nil {
		logrus.WithContext(ctx).Fatalf("failed to init docker client; %v", err)
	}
	defer cli.Close()

	targetAdr, err := docker.NewCifsAddress(target)
	if err != nil {
		logrus.WithContext(ctx).Errorf("failed to parse target address; %v", err)
	}

	target, err := docker.NewCifsVolume(ctx, cli, targetAdr, "backup_volume")
	if err != nil {
		logrus.WithContext(ctx).Errorf("failed to create cifs backup volume; %v", err)
	}
	defer target.Destroy(ctx)

	containers := strings.Split(container, ",")
	sources := make([]*docker.Container, len(containers))

	for i, c := range containers {
		source, err := docker.FindContainerByName(ctx, cli, c)
		if err != nil {
			logrus.WithContext(ctx).Error(err)
			continue
		}

		if hold {
			err = source.Stop(ctx)
			if err != nil {
				logrus.WithContext(ctx).WithField("container", c).Fatalf("failed to stop container; %v", err)
			}
			defer source.Start(ctx)
		}

		sources[i] = source
	}

	for _, s := range sources {
		err := docker.Backup(ctx, cli, s, *target)
		if err != nil {
			logrus.Error(err)
		}
	}
}
