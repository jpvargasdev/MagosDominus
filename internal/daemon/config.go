package daemon

import (
  "log"
  "os"

  "github.com/joho/godotenv"
)


type Config struct {
  RepoURL string
}

func getRepo() *Config {
  err := godotenv.Load()
  if err != nil {
    log.Fatal("Error loading .env file")
  }
  
  return &Config{
    RepoURL: os.Getenv("MD_REPO"),
  }
}

