package main

import (
	"log"

	"mini-gitops/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
