package watcher

import (
	"log"
	"magos-dominus/internal/cli"
	"os"
)

func main() {
	if err := cli.Execute(); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}
