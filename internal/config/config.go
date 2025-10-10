package config 

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)


type Config struct {
  RepoURL        string
  PreferDigest   bool
  AppId          int64
  InstallationId int64 
  PrivateKeyPath string
}

func GetPreferDigest() *Config {
  err := godotenv.Load()
  if err != nil {
    log.Fatal("Error loading .env file")
  }
  
  return &Config{
    PreferDigest: os.Getenv("MD_PREFER_DIGEST") == "true",
  }
}

func GetGithubConfig() *Config {
  err := godotenv.Load()
  if err != nil {
    log.Fatal("Error loading .env file")
  }

  appId, err := strconv.ParseInt(os.Getenv("GH_APP_ID"), 10, 64)
  if err != nil {
    log.Fatal("Error parsing GH_APP_ID")
  }
  
  installationId, err := strconv.ParseInt(os.Getenv("GH_INSTALLATION_ID"), 10, 64)
  if err != nil {
    log.Fatal("Error parsing GH_INSTALLATION_ID")
  }
  
  return &Config{
    RepoURL:        os.Getenv("MD_REPO"),
    AppId:          appId,
    InstallationId: installationId,
    PrivateKeyPath: os.Getenv("GH_PRIVATE_KEY_PATH"),
  }
}
