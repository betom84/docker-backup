package cmd

import (
	"fmt"
	"strings"

	"github.com/betom84/docker-backup/docker"
	"github.com/spf13/cobra"
)

var (
	container string
	target    string
	hold      bool
)

func NewBackupCommand() *cobra.Command {
	backupCmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup Docker container volumes to CIFS network share",
		RunE:  runBackupCmd,
	}

	backupCmd.Flags().StringVar(&container, "container", "", "Comma separated list of Docker container names (required)")
	backupCmd.Flags().StringVar(&target, "target", "", "CIFS network share address like csif://user:pass@host/path (required)")
	backupCmd.Flags().BoolVar(&hold, "hold", false, "Hold containers during backup")

	backupCmd.MarkFlagRequired("container")
	backupCmd.MarkFlagRequired("target")

	return backupCmd
}

func runBackupCmd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	cli, err := docker.NewClient(ctx, host)
	if err != nil {
		return fmt.Errorf("failed to init docker client; %w", err)
	}
	defer cli.Close()

	target, err := docker.NewVolume(ctx, cli, target, "backup_volume")
	if err != nil {
		return fmt.Errorf("failed to init backup volume; %w", err)
	}
	defer target.Destroy(ctx)

	containers := strings.Split(container, ",")
	sources := make([]*docker.Container, 0, len(containers))

	for _, c := range containers {
		source, err := docker.FindContainerByName(ctx, cli, c)
		if err != nil {
			return err
		}

		if hold {
			err = source.Stop(ctx)
			if err != nil {
				return fmt.Errorf("failed to stop container; %w", err)
			}
			defer source.Start(ctx)
		}

		sources = append(sources, source)
	}

	for _, s := range sources {
		err := docker.Backup(ctx, cli, s, target)
		if err != nil {
			return fmt.Errorf("backup failed; %w", err)
		}
	}

	return nil
}
