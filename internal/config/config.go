// Package config provides CLI configuration and application logic for kessoku.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/mazrean/kessoku/internal/kessoku"
	"github.com/mazrean/kessoku/internal/llmsetup"
	"github.com/mazrean/kessoku/internal/migrate"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// CLI is the root command configuration with subcommands.
type CLI struct {
	LogLevel string               `kong:"short='l',help='Log level',enum='debug,info,warn,error',default='info'"`
	Generate GenerateCmd          `kong:"cmd,default='withargs',help='Generate DI code (default)'"`
	Migrate  MigrateCmd           `kong:"cmd,help='Migrate wire config to kessoku'"`
	LLMSetup llmsetup.LLMSetupCmd `kong:"cmd,name='llm-setup',help='Setup coding agent skills'"`
	Version  kong.VersionFlag     `kong:"short='v',help='Show version and exit.'"`
}

// GenerateCmd is the default command for generating DI code.
type GenerateCmd struct {
	Files []string `kong:"arg,help='Go files to process'"`
}

// Run executes the generate command.
func (c *GenerateCmd) Run(cli *CLI) error {
	setupLogger(cli.LogLevel)

	if len(c.Files) == 0 {
		return fmt.Errorf("no files specified")
	}

	slog.Info("Generating dependency injection code", "files", c.Files)

	processor := kessoku.NewProcessor()
	return processor.ProcessFiles(c.Files)
}

// MigrateCmd is the command for migrating wire files to kessoku format.
type MigrateCmd struct {
	Output   string   `kong:"short='o',default='kessoku.go',help='Output file path'"`
	Patterns []string `kong:"arg,optional,help='Go package patterns to migrate',default='./'"`
}

// Run executes the migrate command.
func (c *MigrateCmd) Run(cli *CLI) error {
	setupLogger(cli.LogLevel)

	slog.Info("Migrating wire configuration", "patterns", c.Patterns)

	migrator := migrate.NewMigrator()
	return migrator.MigrateFiles(c.Patterns, c.Output)
}

func Run() error {
	var cli CLI
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

	return kongCtx.Run(&cli)
}

func setupLogger(level string) {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: parseLogLevel(level),
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
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
