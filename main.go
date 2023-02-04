package main

import (
	"context"

	"github.com/betom84/docker-backup/cmd"
	"github.com/sirupsen/logrus"
)

func main() {
	err := cmd.Execute(context.Background())
	if err != nil {
		logrus.Fatal(err)
	}
}
