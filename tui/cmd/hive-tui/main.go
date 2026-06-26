package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"hive-tui/internal/api"
	"hive-tui/internal/app"
	"hive-tui/internal/config"
	"hive-tui/internal/wsclient"
	"hive-tui/internal/wsproto"
)

func main() {
	configPath := flag.String("config", "config.toml", "path to config.toml")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Hive TUI\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n  hive-tui [--config config.toml]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	normalized := cfg.Normalized()

	client := api.NewClient(normalized.RESTBase)
	model := app.NewModel(app.Dependencies{
		Config: normalized,
		API:    client,
		ConnectWS: func(ctx context.Context, token string, events chan<- wsproto.Envelope) (app.WSClient, error) {
			return wsclient.Dial(ctx, normalized.WSBase, token, events)
		},
	})

	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
