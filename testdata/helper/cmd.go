package helper

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/betom84/docker-backup/cmd"
)

func RunDaemonCommand(t *testing.T, opts ...string) error {
	t.Helper()

	deadline := time.Now().Truncate(1 * time.Minute).Add(90 * time.Second)
	t.Logf("%s -> %s", time.Now(), deadline)
	ctx, cancel := context.WithDeadline(context.TODO(), deadline)
	defer cancel()

	cmd := cmd.NewRootCommand()
	cmd.SetOutput(os.Stdout)

	args := []string{"daemon", "--verbosity", "trace"}
	args = append(args, opts...)
	cmd.SetArgs(args)

	return cmd.ExecuteContext(ctx)
}

func RunBackupCommand(t *testing.T, opts ...string) error {
	t.Helper()

	cmd := cmd.NewRootCommand()
	cmd.SetOutput(os.Stdout)

	args := []string{"backup", "--verbosity", "trace"}
	args = append(args, opts...)
	cmd.SetArgs(args)

	return cmd.Execute()
}
