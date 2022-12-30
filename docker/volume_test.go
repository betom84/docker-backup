package docker_test

import (
	"testing"

	"github.com/betom84/docker-backup/docker"
	"github.com/stretchr/testify/assert"
)

func TestNewCifsAddress(t *testing.T) {
	testCases := []struct {
		desc    string
		address string
		result  docker.CifsAddress
		err     error
	}{
		{
			desc:    "successfully parse valid address",
			address: "chuck:secret@myhost/mypath",
			result:  docker.CifsAddress{Host: "myhost", Path: "mypath", Username: "chuck", Password: "secret"},
			err:     nil,
		},
		{
			desc:    "successfully parse address without path",
			address: "chuck:secret@myhost/",
			result:  docker.CifsAddress{Host: "myhost", Path: "", Username: "chuck", Password: "secret"},
			err:     nil,
		},
		{
			desc:    "successfully parse address with special password chars",
			address: `chuck:abcABC123()[]/\*#%$§"_-+.,!?-@myhost/mypath`,
			result:  docker.CifsAddress{Host: "myhost", Path: "mypath", Username: "chuck", Password: `abcABC123()[]/\*#%$§"_-+.,!?-`},
			err:     nil,
		},
		{
			desc:    "successfully parse valid address with nested path",
			address: "chuck:secret@myhost/my/nested/path/",
			result:  docker.CifsAddress{Host: "myhost", Path: "my/nested/path/", Username: "chuck", Password: "secret"},
			err:     nil,
		},
		{
			desc:    "successfully parse valid address with host ip",
			address: "chuck:secret@127.0.0.1/mypath",
			result:  docker.CifsAddress{Host: "127.0.0.1", Path: "mypath", Username: "chuck", Password: "secret"},
			err:     nil,
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			result, err := docker.NewCifsAddress(tC.address)
			assert.Equal(t, tC.result, result)
			assert.Equal(t, tC.err, err)
		})
	}
}
