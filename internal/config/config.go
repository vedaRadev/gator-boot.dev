package config

import (
    "path/filepath"
    "encoding/json"
    "os"
)

const defaultConfigFileName string = ".gatorconfig.json"

type Config struct {
    DbUrl string `json:"db_url"`
    CurrentUserName string `json:"current_user_name"`
}

func Read() (Config, error) {
    var config Config

    home, err := os.UserHomeDir()
    if err != nil { return config, err }

    configData, err := os.ReadFile(filepath.Join(home, defaultConfigFileName))
    if err != nil { return config, err }

    if err := json.Unmarshal(configData, &config); err != nil { return config, err }

    return config, nil
}

// TODO Should this reset to the previous username in case of a failure?
func (c *Config) SetUser(username string) error {
    c.CurrentUserName = username
    home, err := os.UserHomeDir()
    if err != nil { return err }

    file, err := os.Create(filepath.Join(home, defaultConfigFileName))
    defer file.Close()
    if err != nil { return err }

    if err := json.NewEncoder(file).Encode(c); err != nil { return err }

    return nil
}
