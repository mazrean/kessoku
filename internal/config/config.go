// Package config provides CLI configuration and application logic for kessoku.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/alecthomas/kong"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type Config struct {
	LogLevel string           `kong:"short='l',help='Log level',enum='debug,info,warn,error',default='info'"`
	Name     string           `kong:"arg,optional,help='Name to greet',default='World'"`
	Version  kong.VersionFlag `kong:"short='v',help='Show version and exit.'"`
}

func Run() error {
	var cli Config
	kongCtx := kong.Parse(&cli,
		kong.Name("kessoku"),
		kong.Description("A CLI tool for managing kessoku"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"version": fmt.Sprintf("%s (%s) released on %s", version, commit, date),
		},
	)

	return kongCtx.Run()
}

func (c *Config) Run() error {
	// Setup slog with TextHandler
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: c.parseLogLevel(),
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Debug("Starting application", "name", c.Name, "log_level", c.LogLevel)

	fmt.Printf("Hello, %s!\n", c.Name)

	slog.Debug("Application completed successfully")
	return nil
}

func (c *Config) parseLogLevel() slog.Level {
	switch strings.ToLower(c.LogLevel) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
