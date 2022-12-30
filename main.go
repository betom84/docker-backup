package main

import (
	"fmt"
	"os"

	"github.com/betom84/docker-backup/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
