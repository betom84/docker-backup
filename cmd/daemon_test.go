package cmd_test

import (
	"fmt"
	"testing"

	"github.com/betom84/docker-backup/docker"
	"github.com/stretchr/testify/assert"

	"github.com/betom84/docker-backup/testdata/helper"
)

func TestDaemonCommand_Single(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	localBackupFolder := t.TempDir()

	bb := helper.NewBusyboxContainer(t, "chuck", map[string]string{
		string(docker.Enabled):  "true",
		string(docker.Target):   fmt.Sprintf("local://%s", localBackupFolder),
		string(docker.Schedule): "*/1 * * * *",
	})

	err := helper.RunDaemonCommand(t)
	assert.NoError(t, err)

	helper.AssertBackupFilesExists(t, localBackupFolder, bb)
}

func TestDaemonCommand_Group(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	localBackupFolder := t.TempDir()

	bb := []helper.BusyboxContainer{
		helper.NewBusyboxContainer(t, "tick", map[string]string{
			string(docker.Enabled): "true",
			string(docker.Group):   "ducktales",
		}),
		helper.NewBusyboxContainer(t, "trick", map[string]string{
			string(docker.Enabled): "true",
			string(docker.Group):   "ducktales",
		}),
		helper.NewBusyboxContainer(t, "track", map[string]string{
			string(docker.Enabled): "true",
			string(docker.Group):   "ducktales",
		}),
	}

	err := helper.RunDaemonCommand(t, "--defaultTarget", fmt.Sprintf("local://%s", localBackupFolder), "--defaultSchedule", "*/1 * * * *")
	assert.NoError(t, err)

	helper.AssertBackupFilesExists(t, localBackupFolder, bb...)
}
