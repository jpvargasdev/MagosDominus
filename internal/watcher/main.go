package main

import (
	"log"
)

func main() {
	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
