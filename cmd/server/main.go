package main 

import (
	"log"

	"github.com/jpvargasdev/magos-dominus/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
