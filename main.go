package main

import _ "github.com/lib/pq"

import (
    "fmt"
    "os"
    "database/sql"
    "context"
    "time"

    "github.com/google/uuid"

    "github.com/vedaRadev/gator-boot.dev/internal/database"
    "github.com/vedaRadev/gator-boot.dev/internal/config"
)

type CommandMap map[string]func(*State, []string)error
type State struct   {
    Cfg *config.Config
    Db *database.Queries
}

// TODO Get rid of as many global variables as possible
var Commands CommandMap

func HandleLogin(s *State, args []string) error {
    if len(args) == 0 { return fmt.Errorf("expected argument: username") }
    user, err := s.Db.GetUser(context.Background(), args[0])
    if err != nil {
        return fmt.Errorf(
            "Failed to log in. Does the user exist? Error: %w",
            err,
        )
    }

    if err := s.Cfg.SetUser(user.Name); err != nil {
        return fmt.Errorf(
            "The user was retrieved from the DB but we failed to write them to the gatorconfig: %w",
            err,
        )
    }

    return nil
}

// TODO Do we really want to set the new user as the current immediately upon registration?
func HandleRegister(s *State, args []string) error {
    if len(args) == 0 { return fmt.Errorf("expected argument: name") }

    now := time.Now()
    params := database.CreateUserParams {
        ID: uuid.New(),
        CreatedAt: now,
        UpdatedAt: now,
        Name: args[0],
    }

    user, err := s.Db.CreateUser(context.Background(), params)
    if err != nil { return err }

    if err := s.Cfg.SetUser(user.Name); err != nil {
        fmt.Printf(
            "WARNING: User was created in the db but we failed to write them to the gatorconfig: %v\n",
            err,
        )
    }

    return nil
}

func HandleReset(s *State, args []string) error {
    _, err := s.Db.Reset(context.Background())
    if err != nil { return err }

    return nil
}

func HandleUsers(s *State, args []string) error {
    users, err := s.Db.GetUsers(context.Background())
    if err != nil { return err }

    for _, user := range users {
        fmt.Printf("* %v", user.Name)
        if user.Name == s.Cfg.CurrentUserName {
            fmt.Print(" (current)")
        }
        fmt.Println()
    }

    return nil
}

func main() {
    var state State

    cfg, err := config.Read()
    if err != nil {
        fmt.Printf("Failed to get gator config: %v\n", err)
        os.Exit(1)
    }
    state.Cfg = &cfg

    db, err := sql.Open("postgres", state.Cfg.DbUrl)
    if err != nil {
        fmt.Println("Failed to open connection to the db url specific in gatorconfig")
        os.Exit(1)
    }
    state.Db = database.New(db)


    Commands = make(CommandMap)
    Commands["login"] = HandleLogin
    Commands["register"] = HandleRegister
    Commands["reset"] = HandleReset
    Commands["users"] = HandleUsers

    // NOTE do we want to slice args or just pass the entire os args through to every command?
    args := os.Args[1:]
    if len(args) == 0 {
        // TODO print help info
        fmt.Printf("Expected command, one of:\n");
        for key := range Commands {
            fmt.Printf("* %v\n", key)
        }

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
