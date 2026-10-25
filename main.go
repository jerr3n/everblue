package main

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.DisableStacktrace = true
	logger, _ := cfg.Build()
	home, err := os.UserHomeDir() // /home/user
	if err != nil {
		logger.Fatal("home directory err", zap.Error(err))
	}
	logger.Debug("home directory ready", zap.String("dir", home))
	root := &cli.Command{
		Name:  "everblue",
		Usage: "Launch Roblox Studio",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			runLauncher(home, logger)
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:    "setup",
				Aliases: []string{"init", "initialize"},
				Usage:   "Set up Everblue",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					setup(home, logger)
					return nil
				},
			},
			{
				Name:  "install",
				Usage: "Install a Studio Instance",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return installStudio(ctx, home, logger, nil)
				},
			},
			{
				Name:  "run",
				Usage: "Run Studio",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return runStudio(ctx, home, logger)
				},
			},
			{
				Name:    "cfg",
				Aliases: []string{"config"},
				Usage:   "Open the Everblue window",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					runConfig(home, logger)
					return nil
				},
			},
		},
	}
	if err := root.Run(context.Background(), os.Args); err != nil {
		logger.Fatal("error", zap.Error(err))
	}
}
