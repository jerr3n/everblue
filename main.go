package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jerr3n/everblue/util"
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
		Name: "everblue",
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
					setup(home, logger)
					data, _ := util.GetVersion(logger)
					_ = util.DownloadVersion(logger, *data, home)
					util.RunRosemary(logger, util.RosemaryOptions{}, []string{"run", fmt.Sprintf("%s/.local/share/everblue/versions/latest/RobloxStudioInstaller.app", home)})
					return nil
				},
			},
			{
				Name:  "run",
				Usage: "Run Studio",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					// ~/.local/share/rosemary/prefixes/default/
					util.RunRosemary(logger, util.RosemaryOptions{}, []string{"run", fmt.Sprintf("%s/.local/share/rosemary/prefixes/default/root/Applications/RobloxStudio.app", home)})
					return nil
				},
			},
		},
	}
	if err := root.Run(context.Background(), os.Args); err != nil {
		logger.Fatal("error", zap.Error(err))
	}
}
