package main

import (
    "fmt"
    "os"

    "github.com/vedaRadev/gator-boot.dev/internal/config"
)

type State struct { cfg *config.Config }
type CommandMap map[string]func(*State, []string)error

// TODO Get rid of as many global variables as possible
var Commands CommandMap

func HandleLogin(s *State, args []string) error {
    if len(args) == 0 { return fmt.Errorf("username argument is required") }
    if err := s.cfg.SetUser(args[0]); err != nil { return err }
    return nil
}

func main() {
    cfg, err := config.Read()
    if err != nil {
        fmt.Printf("Failed to get gator config: %v\n", err)
        os.Exit(1)
    }
    var state State
    state.cfg = &cfg

    Commands = make(CommandMap)
    Commands["login"] = HandleLogin

    // NOTE do we want to slice args or just pass the entire os args through to every command?
    args := os.Args[1:]
    if len(args) == 0 {
        // TODO print help info
        fmt.Printf("Expected command\n");
        os.Exit(1)
    }

    commandName := args[0]
    if handler, exists := Commands[commandName]; exists {
        if err := handler(&state, args[1:]); err != nil {
            fmt.Printf("Failed: %v\n", err)
            os.Exit(1)
        }
    } else {
        fmt.Println("Unrecognized command")
        os.Exit(1)
    }
}
