package main

import (
	"log"
	"magos-dominus/internal/cli"
)

func main() {
	if err := internal.Execute(); err != nil {
		log.Fatal(err)
	}
}
