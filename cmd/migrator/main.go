package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

// Set at build time: -ldflags "-X main.version=v0.1.0 -X main.commit=abc1234"
var (
	version = "dev"
	commit  = "none"
)

type globalFlags struct {
	logLevel  string
	logFormat string
	noColor   bool
}

func main() {
	// A second interrupt kills the process outright: the first one asks the
	// pipeline to stop and checkpoint, and someone who presses Ctrl-C twice
	// wants out now, not a graceful drain of a 100M-row table.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := newRootCmd().ExecuteContext(ctx); err != nil {
		// Cobra has already printed usage errors; everything else is ours.
		if !errors.Is(err, errSilent) {
			fmt.Fprintln(os.Stderr, "error: "+err.Error())
		}
		os.Exit(1)
	}
}

// errSilent marks an error whose message has already been reported.
var errSilent = errors.New("silent")

func newRootCmd() *cobra.Command {
	var g globalFlags

	cmd := &cobra.Command{
		Use:   "migrator",
		Short: "Bulk data migration between databases",
		Long: "migrator copies data in bulk between databases — MongoDB, PostgreSQL,\n" +
			"MySQL and SQLite, in any direction — then exits.\n\n" +
			"Start with `migrator doctor` to see what a connection supports, and\n" +
			"`migrator plan` to see what a job would do before it does it.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return setupLogging(g)
		},
	}

	f := cmd.PersistentFlags()
	f.StringVar(&g.logLevel, "log-level", "info", "log level: debug, info, warn, error")
	f.StringVar(&g.logFormat, "log-format", "text", "log format: text, json")
	f.BoolVar(&g.noColor, "no-color", false, "disable coloured output")

	cmd.AddCommand(
		newValidateCmd(),
		newVersionCmd(),
	)
	return cmd
}

func setupLogging(g globalFlags) error {
	var level slog.Level
	switch g.logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		return fmt.Errorf("unknown --log-level %q (want debug, info, warn or error)", g.logLevel)
	}

	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	switch g.logFormat {
	case "text":
		h = slog.NewTextHandler(os.Stderr, opts)
	case "json":
		h = slog.NewJSONHandler(os.Stderr, opts)
	default:
		return fmt.Errorf("unknown --log-format %q (want text or json)", g.logFormat)
	}

	slog.SetDefault(slog.New(h))
	return nil
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "migrator %s (%s)\n", version, commit)
			return nil
		},
	}
}
