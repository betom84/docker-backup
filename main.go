package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/betom84/docker_backup/docker"
	"github.com/betom84/docker_backup/utils"
)

var (
	host      = flag.String("host", "", "TCP Docker host (port 2375)")
	container = flag.String("container", "", "Comma separated list of Docker container names")
	target    = flag.String("target", "", "CIFS volume address (user:pass@host/path)")
)

func main() {
	flag.Parse()
	if utils.IsEmpty(*host, *container, *target) {
		fmt.Printf("usage: %s [options]\n\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Printf("\nexample: ./docker_backup -host docker.local -target user:secret@mynas.local/backups -container container1,container2\n")
		os.Exit(1)
	}

	ctx := context.Background()

	cli, err := docker.NewClient(*host)
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	targetAdr, err := docker.NewCifsAddress(*target)
	if err != nil {
		panic(err)
	}

	target, err := docker.NewCifsVolume(ctx, cli, targetAdr, "backup_volume")
	if err != nil {
		panic(err)
	}
	defer target.Destroy(ctx)

	for _, c := range strings.Split(*container, ",") {
		err := runContainerBackup(ctx, cli, *target, c)
		if err != nil {
			fmt.Printf("container backup failed; %s\n", err)
		}
	}
}

func runContainerBackup(ctx context.Context, cli docker.Client, target docker.Volume, container string) error {
	source, err := docker.FindContainer(ctx, cli, container)
	if err != nil {
		return err
	}

	backup, err := docker.NewBackupContainer(ctx, source, target)
	if err != nil {
		return err
	}
	defer backup.Destroy(ctx)

	err = backup.Start(ctx)
	if err != nil {
		return err
	}

	err = backup.Backup(ctx)
	if err != nil {
		return err
	}

	return nil
}
