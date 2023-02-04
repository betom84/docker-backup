package docker

import (
	"context"
	"fmt"
	"net/url"

	dv "github.com/docker/docker/api/types/volume"
	"github.com/sirupsen/logrus"
)

type Volume interface {
	Create(context.Context) error
	Destroy(context.Context) error
	String() string
	GetName() string
	URL() *url.URL
}

func NewVolume(ctx context.Context, client Client, target string, name string) (Volume, error) {
	var err error
	var v Volume

	t, err := url.ParseRequestURI(target)
	if err != nil {
		return nil, err
	}

	switch t.Scheme {
	case "local":
		v = volume{name: t.Path, url: t}
	case "cifs":
		v = cifsVolume{volume{name: name, url: t, client: client}}
		logrus.WithContext(ctx).WithField("volume", v.String()).Debugln("cifs volume created")
	default:
		err = fmt.Errorf("unsupported url scheme")
	}

	return v, err
}

type volume struct {
	name   string
	url    *url.URL
	client Client
}

func (v volume) Create(ctx context.Context) error {
	return nil
}
func (v volume) Destroy(ctx context.Context) error {
	return nil
}

func (v volume) String() string {
	return fmt.Sprintf("%s (%s)", v.name, v.url.String())
}

func (v volume) GetName() string {
	return v.name
}

func (v volume) URL() *url.URL {
	return v.url
}

type cifsVolume struct {
	volume
}

func (v cifsVolume) Create(ctx context.Context) error {
	username := v.url.User.Username()
	password, _ := v.url.User.Password()

	_, err := v.client.VolumeCreate(ctx, dv.VolumeCreateBody{
		Driver: "local",
		DriverOpts: map[string]string{
			"device": fmt.Sprintf("//%s/%s", v.url.Host, v.url.Path),
			"o":      fmt.Sprintf("addr=%s,username=%s,password=%s,file_mode=0777,dir_mode=0777", v.url.Host, username, password),
			"type":   "cifs",
		},
		Labels: map[string]string{},
		Name:   v.name,
	})

	return err
}

func (v cifsVolume) Destroy(ctx context.Context) error {
	err := v.client.VolumeRemove(ctx, v.name, false)

	logrus.WithContext(ctx).WithFields(logrus.Fields{
		"volume": v.String(),
		"error":  err,
	}).Debugln("cifs volume destroyed")

	return err
}
