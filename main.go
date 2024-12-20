package main

import (
    "fmt"

    "github.com/vedaRadev/gator-boot.dev/internal/config"
)

func main() {
    cfg, err := config.Read()
    if err != nil {
        fmt.Printf("Failed to get gator config: %v\n", err)
        return
    }

    cfg, err = config.Read()
    if err != nil {
        fmt.Printf("Failed to get gator config: %v\n", err)
        return
    }

    fmt.Println(cfg.DbUrl)
    fmt.Println(cfg.CurrentUserName)
}
