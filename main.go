package main

import _ "github.com/lib/pq"

import (
    "fmt"
    "os"
    "database/sql"
    "context"
    "time"
    "net/http"
    "io"
    "encoding/xml"
    "bytes"
    "html"
    "strings"

    "github.com/google/uuid"

    "github.com/vedaRadev/gator-boot.dev/internal/database"
    "github.com/vedaRadev/gator-boot.dev/internal/config"
)

// TODO Is this app state or command state? Not sure yet.
// Commands need access to the State type regardless.
type State struct   {
    Cfg *config.Config
    Db *database.Queries
}

// TODO move commands to their own dir?
//============================== COMMANDS ==============================// 
type CommandMap map[string]func(*State, []string)error
var Commands CommandMap

func middlewareLoggedIn(handler func(*State, []string, database.User) error) func(*State, []string) error {
    return func(s *State, args []string) error {
        currentUser, err := s.Db.GetUser(context.Background(), s.Cfg.CurrentUserName)
        if err != nil { return fmt.Errorf("failed to get current user from db: %w", err) }
        return handler(s, args, currentUser)
    }
}

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

func HandleAgg(s *State, args []string) error {
    // if len(args) == 0 { return fmt.Errorf("expected argument: rss url") }
    // feed, err := FetchFeed(context.Background(), args[0])
    feed, err := FetchFeed(context.Background(), "https://wagslane.dev/index.xml")
    if err != nil { return err }
    fmt.Printf("feed: %v\n", feed)
    return nil
}

func followFeed(s *State, userId uuid.UUID, feedId uuid.UUID) (database.CreateFeedFollowRow, error) {
    var feedFollow database.CreateFeedFollowRow

    now := time.Now()
    params := database.CreateFeedFollowParams {
        ID: uuid.New(),
        CreatedAt: now,
        UpdatedAt: now,
        UserID: userId,
        FeedID: feedId,
    }

    feedFollow, err := s.Db.CreateFeedFollow(context.Background(), params)
    if err != nil { return feedFollow, err }

    return feedFollow, nil
}

func HandleAddFeed(s *State, args []string, currentUser database.User) error {
    if len(args) != 2 { return fmt.Errorf("expected 2 arguments: feed_name feed_url") }

    now := time.Now()
    params := database.CreateFeedParams {
        ID: uuid.New(),
        CreatedAt: now,
        UpdatedAt: now,
        Name: args[0],
        Url: args[1],
        UserID: currentUser.ID,
    }

    feed, err := s.Db.CreateFeed(context.Background(), params)
    if err != nil { return fmt.Errorf("failed to create and insert feed: %w", err) }

    fmt.Printf("Added feed: %v\n", feed)

    feedFollow, err := followFeed(s, currentUser.ID, feed.ID)
    if err != nil { return fmt.Errorf("failed to follow feed: %w", err) }

    fmt.Printf("%v has followed feed %v\n", feedFollow.UserName, feedFollow.FeedName);

    return nil
}

func HandleFeeds(s *State, args []string) error {
    feeds, err := s.Db.GetFeeds(context.Background())
    if err != nil { return fmt.Errorf("Failed to get feeds from db: %w", err) }

    for _, feed := range feeds {
        fmt.Printf("%v [%v] (%v)\n", feed.Name, feed.Url, feed.UserName)
    }

    return nil
}

func HandleFollow(s *State, args []string, currentUser database.User) error {
    if len(args) != 1 { return fmt.Errorf("expected 1 argument: feed_url") }

    feed, err := s.Db.GetFeed(context.Background(), args[0])
    if err != nil { return fmt.Errorf("failed to get feed from db (do you need to create it?): %w", err) }

    feedFollow, err := followFeed(s, currentUser.ID, feed.ID)
    if err != nil { return fmt.Errorf("failed to follow feed (are you already following it?): %w", err) }
    
    fmt.Printf("%v has followed feed %v\n", feedFollow.UserName, feedFollow.FeedName);

    return nil
}

func HandleFollowing(s *State, args []string, currentUser database.User) error {
    if len(args) > 0 { return fmt.Errorf("expected 0 arguments") }

    feeds, err := s.Db.GetFeedFollowsForUser(context.Background(), currentUser.ID)
    if err != nil { return fmt.Errorf("failed to get feed follows: %w", err) }

    if len(feeds) > 0 {
        fmt.Printf("%v is following\n", currentUser.Name)
        for _, feed := range feeds {
            fmt.Printf(" - %v (%v)\n", feed.FeedName, feed.FeedUrl)
        }
    } else {
        fmt.Printf("%v is not following any feeds\n", currentUser.Name)
    }

    return nil
}
//============================== END COMMANDS ==============================// 
type RssFeed struct {
    Channel struct {
        Title string `xml:"title"`
        Link string `xml:"link"`
        Description string `xml:"description"`
        Item []RssItem `xml:"item"`
    } `xml:"channel"`
}

type RssItem struct {
    Title string `xml:"title"`
    Link string `xml:"link"`
    Description string `xml:"description"`
    PubDate string `xml:"pubDate"`
}

func FetchFeed(ctx context.Context, feedUrl string) (*RssFeed, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", feedUrl, bytes.NewBuffer([]byte {}))
    if err != nil { return nil, err }
    req.Header.Set("User-Agent", "gator")

    res, err := (&http.Client {}).Do(req)
    defer res.Body.Close()
    if err != nil { return nil, err }

    data, err := io.ReadAll(res.Body)
    if err != nil { return nil, err }

    var rssFeed RssFeed
    if err := xml.Unmarshal(data, &rssFeed); err != nil { return nil, err }

    rssFeed.Channel.Title = html.UnescapeString(rssFeed.Channel.Title)
    rssFeed.Channel.Description = html.UnescapeString(rssFeed.Channel.Description)
    for i := range rssFeed.Channel.Item {
        item := &rssFeed.Channel.Item[i]
        item.Title = html.UnescapeString(item.Title)
        item.Description = html.UnescapeString(item.Description)
    }

    return &rssFeed, nil
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
    Commands["agg"] = HandleAgg
    Commands["addfeed"] = middlewareLoggedIn(HandleAddFeed)
    Commands["feeds"] = HandleFeeds
    Commands["follow"] = middlewareLoggedIn(HandleFollow)
    Commands["following"] = middlewareLoggedIn(HandleFollowing)

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
    if handler, exists := Commands[strings.ToLower(commandName)]; exists {
        if err := handler(&state, args[1:]); err != nil {
            fmt.Printf("Failed: %v\n", err)
            os.Exit(1)
        }
    } else {
        fmt.Println("Unrecognized command")
        os.Exit(1)
    }
}
