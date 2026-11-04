package main

import (
	"fmt"
	"log/slog"
	"os"

	"hr-system/internal/app"
	"hr-system/internal/config"
	"hr-system/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "load config:", err)
		os.Exit(1)
	}

	log, closeLog, err := logger.New(logger.Config{Env: cfg.App.Env, Level: cfg.App.LogLevel, File: cfg.App.LogFile})
	if err != nil {
		fmt.Fprintln(os.Stderr, "create logger:", err)
		os.Exit(1)
	}

	slog.SetDefault(log)

	runErr := app.Run(cfg)
	if runErr != nil {
		slog.Error("application stopped with error", "error", runErr)
	}

	if err := closeLog(); err != nil {
		fmt.Fprintln(os.Stderr, "close log file:", err)
	}
	if runErr != nil {
		fmt.Fprintln(os.Stderr, "application error:", runErr)
		os.Exit(1)
	}
}
