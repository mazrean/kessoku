// Package config provides CLI configuration and application logic for kessoku.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/mazrean/kessoku/internal/kessoku"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type Config struct {
	LogLevel string           `kong:"short='l',help='Log level',enum='debug,info,warn,error',default='info'"`
	Files    []string         `kong:"arg,help='Go files to process'"`
	Version  kong.VersionFlag `kong:"short='v',help='Show version and exit.'"`
}

func (c *Config) Run() error {
	// Setup slog with TextHandler
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: parseLogLevel(c.LogLevel),
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	if len(c.Files) == 0 {
		return fmt.Errorf("no files specified")
	}

	slog.Info("Generating dependency injection code", "files", c.Files)

	processor := kessoku.NewProcessor()
	return processor.ProcessFiles(c.Files)
}

func Run() error {
	var cli Config
	kongCtx := kong.Parse(&cli,
		kong.Name("kessoku"),
		kong.Description("A dependency injection code generator for Go, similar to google/wire"),
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

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
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
