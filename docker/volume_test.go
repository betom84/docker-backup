package docker

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewVolume(t *testing.T) {
	testCases := []struct {
		desc               string
		address            string
		expectedVolumeURL  url.URL
		expectedVolumeName string
		err                error
	}{
		{
			desc:               "successfully parse valid local address",
			address:            "local:///mypath",
			expectedVolumeURL:  url.URL{Scheme: "local", Host: "", Path: "/mypath"},
			expectedVolumeName: "/mypath",
			err:                nil,
		},
		{
			desc:              "successfully parse valid cifs address",
			address:           "cifs://chuck:secret@myhost/mypath",
			expectedVolumeURL: url.URL{Scheme: "cifs", Host: "myhost", Path: "/mypath", User: url.UserPassword("chuck", "secret")},
			err:               nil,
		},
		{
			desc:              "successfully parse cifs address without path",
			address:           "cifs://chuck:secret@myhost/",
			expectedVolumeURL: url.URL{Scheme: "cifs", Host: "myhost", Path: "/", User: url.UserPassword("chuck", "secret")},
			err:               nil,
		},
		{
			desc:              "failed to parse plain cifs address with special password chars",
			address:           `cifs://chuck:abcABC123()[]/\*#%$§"_-+.,!?-@myhost/mypath`,
			expectedVolumeURL: url.URL{Scheme: "cifs", Host: "myhost", Path: "/mypath", User: url.UserPassword("chuck", `abcABC123()[]/\*#%$§"_-+.,!?-`)},
			err:               errors.New("invalid port \":abcABC123()[]\" after host"),
		},
		{
			desc:              "successfully parse encoded cifs address with special password chars",
			address:           fmt.Sprintf("cifs://chuck:%s@myhost/mypath", `abcABC123()[]/\*#%$§"_-+.,!?-`),
			expectedVolumeURL: url.URL{Scheme: "cifs", Host: "myhost", Path: "/mypath", User: url.UserPassword("chuck", `abcABC123()[]/\*#%$§"_-+.,!?-`)},
			err:               errors.New("invalid port \":abcABC123()[]\" after host"),
		},
		{
			desc:              "successfully parse valid cifs address with nested path",
			address:           "cifs://chuck:secret@myhost/my/nested/path/",
			expectedVolumeURL: url.URL{Scheme: "cifs", Host: "myhost", Path: "/my/nested/path/", User: url.UserPassword("chuck", "secret")},
			err:               nil,
		},
		{
			desc:              "successfully parse valid cifs address with host ip",
			address:           "cifs://chuck:secret@127.0.0.1/mypath",
			expectedVolumeURL: url.URL{Scheme: "cifs", Host: "127.0.0.1", Path: "/mypath", User: url.UserPassword("chuck", "secret")},
			err:               nil,
		},
		{
			desc:              "successfully parse valid cifs address with port",
			address:           "cifs://chuck:secret@localhost:6445/mypath",
			expectedVolumeURL: url.URL{Scheme: "cifs", Host: "localhost:6445", Path: "/mypath", User: url.UserPassword("chuck", "secret")},
			err:               nil,
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			v, err := NewVolume(context.TODO(), Client{}, tC.address, "NewVolume")
			e := assert.Equal(t, tC.err, errors.Unwrap(err))
			if !e {
				t.FailNow()
			}

			if tC.err != nil {
				return
			}

			assert.Equal(t, tC.expectedVolumeURL.String(), v.URL().String())

			expPass, _ := tC.expectedVolumeURL.User.Password()
			pass, _ := v.URL().User.Password()
			assert.Equal(t, expPass, pass)

			if tC.expectedVolumeName != "" {
				assert.Equal(t, tC.expectedVolumeName, v.GetName())
			}
		})
	}
}
