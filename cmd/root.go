package cmd

import (
	"fmt"

	"github.com/betom84/docker-backup/docker"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	host      string
	verbosity string
)

var rootCmd = &cobra.Command{
	Use:   "docker-backup",
	Short: "Manage volume backups for Docker hosts.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initLogger(cmd, args)
	},
}

func Execute() error {
	rootCmd.PersistentFlags().StringVarP(&verbosity, "verbosity", "v", logrus.InfoLevel.String(), fmt.Sprintf("%s", logrus.AllLevels))
	rootCmd.PersistentFlags().StringVar(&host, "host", docker.DefaultDockerHost, "Docker host, e.g. tcp://docker.host:2735")

	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(daemonCmd)

	return rootCmd.Execute()
}

func initLogger(cmd *cobra.Command, args []string) error {
	lvl, err := logrus.ParseLevel(verbosity)
	if err == nil {
		logrus.SetLevel(lvl)
	}

	formatter := new(logrus.TextFormatter)
	formatter.TimestampFormat = "02-01-2006 15:04:05"
	formatter.FullTimestamp = true
	logrus.SetFormatter(formatter)

	return nil
}
