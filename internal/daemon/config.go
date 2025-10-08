package daemon

import (
  "log"
  "os"

  "github.com/joho/godotenv"
)


type Config struct {
  RepoURL      string
  PreferDigest bool
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

func getPreferDigest() *Config {
  err := godotenv.Load()
  if err != nil {
    log.Fatal("Error loading .env file")
  }
  
  return &Config{
    PreferDigest: os.Getenv("MD_PREFER_DIGEST") == "true",
  }
}
