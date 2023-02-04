package cmd_test

import (
	"fmt"
	"testing"

	"github.com/betom84/docker-backup/testdata/helper"
	"github.com/stretchr/testify/assert"
)

type AnyError string

func (e AnyError) Error() string {
	return string(e)
}
func (e AnyError) Is(target error) bool {
	return string(e) == target.Error()
}

func TestBackupCommand(t *testing.T) {
	tempDir := t.TempDir()
	bb := helper.NewBusyboxContainer(t, "chuck", map[string]string{})

	tt := []struct {
		desc string
		args []string
		err  error
	}{
		{
			desc: "missing container arg",
			args: []string{"--target", fmt.Sprintf("local://%s", tempDir)},
			err:  AnyError(`required flag(s) "container" not set`),
		},
		{
			desc: "missing target arg",
			args: []string{"--container", "chuck"},
			err:  AnyError(`required flag(s) "target" not set`),
		},
		{
			desc: "invalid container arg",
			args: []string{"--container", "chucky", "--target", fmt.Sprintf("local://%s", tempDir)},
			err:  AnyError("container 'chucky' not found"),
		},
		{
			desc: "invalid target arg",
			args: []string{"--container", "chuck", "--target", tempDir},
			err:  AnyError("failed to init backup volume; unsupported url scheme"),
		},
		{
			desc: "successfully backup container volume to local folder",
			args: []string{"--container", "chuck", "--target", fmt.Sprintf("local://%s", tempDir)},
			err:  nil,
		},
	}

	for _, tc := range tt {
		t.Run(tc.desc, func(t *testing.T) {
			err := helper.RunBackupCommand(t, tc.args...)
			if tc.err == nil {
				assert.NoError(t, err)
				helper.AssertBackupFilesExists(t, tempDir, bb)
			} else {
				assert.ErrorIs(t, tc.err, err)
			}
		})
	}
}
