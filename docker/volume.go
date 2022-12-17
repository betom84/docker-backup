package docker

import (
	"context"
	"fmt"
	"regexp"

	"github.com/docker/docker/api/types/volume"
)

type Volume struct {
	Name     string
	Resource string
	dClient  Client
}

type CifsAddress struct {
	Host     string
	Path     string
	Username string
	Password string
}

func (a CifsAddress) String() string {
	return fmt.Sprintf("%s:*****@%s/%s", a.Username, a.Host, a.Path)
}

func NewCifsAddress(address string) (CifsAddress, error) {
	r, err := regexp.Compile(`^(.*)[:](.*)[@]([a-zA-Z0-9.]*)[/](.*)$`)
	if err != nil {
		return CifsAddress{}, err
	}

	m := r.FindStringSubmatch(address)
	if m == nil || len(m) < 5 {
		return CifsAddress{}, fmt.Errorf("invalid cifs address format; %s", m)
	}

	return CifsAddress{Username: m[1], Password: m[2], Host: m[3], Path: m[4]}, nil
}

func NewCifsVolume(ctx context.Context, client Client, adr CifsAddress, name string) (*Volume, error) {
	_, err := client.VolumeCreate(ctx, volume.VolumeCreateBody{
		Driver: "local",
		DriverOpts: map[string]string{
			"device": fmt.Sprintf("//%s/%s", adr.Host, adr.Path),
			"o":      fmt.Sprintf("addr=%s,username=%s,password=%s,file_mode=0777,dir_mode=0777,vers=1.0", adr.Host, adr.Username, adr.Password),
			"type":   "cifs",
		},
		Labels: map[string]string{},
		Name:   name,
	})

	if err != nil {
		return nil, err
	}

	return &Volume{Name: name, Resource: adr.String(), dClient: client}, nil
}

func (v Volume) Destroy(ctx context.Context) error {
	return v.dClient.VolumeRemove(ctx, v.Name, false)
}
