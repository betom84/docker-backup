package helper

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func AssertBackupFilesExists(t *testing.T, folder string, containers ...BusyboxContainer) {
	t.Helper()

	if len(containers) == 0 {
		return
	}

	for _, c := range containers {
		e, err := os.ReadDir(path.Join(folder, c.Host, c.Name))
		assert.NoError(t, err)

		found := false
		r := regexp.MustCompile(fmt.Sprintf("%s_([0-9]{6})_%s.tar.gz", time.Now().Format("20060102"), c.VolumeName))

		for _, f := range e {
			if found = r.MatchString(f.Name()); found {
				break
			}
		}

		assert.Truef(t, found, "backup file for container %s does not exist", c.Name)
	}

}
