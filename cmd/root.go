package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/betom84/docker-backup/docker"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	host      string
	verbosity string
)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "docker-backup",
		Short:            "Manage volume backups for Docker hosts.",
		SilenceUsage:     true,
		PersistentPreRun: persistentPreRun,
	}

	cmd.PersistentFlags().StringVarP(&verbosity, "verbosity", "v", logrus.InfoLevel.String(), fmt.Sprintf("%s", logrus.AllLevels))
	cmd.PersistentFlags().StringVar(&host, "host", docker.DefaultDockerHost, "Docker host, e.g. tcp://docker.host:2735")

	cmd.AddCommand(NewBackupCommand())
	cmd.AddCommand(NewDaemonCommand())

	return cmd
}

func Execute(ctx context.Context) error {
	cmd := NewRootCommand()

	cmdCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT)
	signal.Notify(sigChan, syscall.SIGTERM)

	go func() {
		select {
		case s := <-sigChan:
			logrus.WithContext(cmd.Context()).Infof("abort (%s)", s)
			cancel()
			return
		case <-cmdCtx.Done():
			return
		}
	}()

	return cmd.ExecuteContext(ctx)
}

func persistentPreRun(cmd *cobra.Command, args []string) {
	lvl, err := logrus.ParseLevel(verbosity)
	if err == nil {
		logrus.SetLevel(lvl)
		logrus.SetReportCaller(lvl == logrus.TraceLevel)
	}

	formatter := new(logrus.TextFormatter)
	formatter.TimestampFormat = "02-01-2006 15:04:05"
	formatter.FullTimestamp = true
	logrus.SetFormatter(formatter)
}
