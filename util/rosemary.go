package util

import (
	"context"

	"github.com/jerr3n/thyme"
	"github.com/jerr3n/thyme/cmd"
	"github.com/jerr3n/thyme/etc"
	"github.com/jerr3n/thyme/prefix"
	"go.uber.org/zap"
	"go.uber.org/zap/zapio"
)

// rosemaryIO sends rosemary's output to the logger and pins it to pfx
func rosemaryIO(log *zap.Logger, pfx prefix.Prefix) (cmd.RunIO, func(), error) {
	dir, err := pfx.Path()
	if err != nil {
		return cmd.RunIO{}, nil, err
	}
	stdout := &zapio.Writer{Log: log.Named("rosemary"), Level: zap.InfoLevel}
	stderr := &zapio.Writer{Log: log.Named("rosemary"), Level: zap.ErrorLevel}
	closeIO := func() {
		stdout.Close()
		stderr.Close()
	}
	// rosemary picks its prefix from the environment, set it so it can't differ from the one we checked
	return cmd.RunIO{Stdout: stdout, Stderr: stderr, Env: []string{"ROSEMARY_PREFIX=" + dir}}, closeIO, nil
}

// RunRosemary runs a Mach-O executable or .app bundle and its arguments in pfx
func RunRosemary(ctx context.Context, log *zap.Logger, pfx prefix.Prefix, options cmd.RosemaryOptions, args []string) error {
	runIO, closeIO, err := rosemaryIO(log, pfx)
	if err != nil {
		log.Error("failed to resolve prefix", zap.Error(err), zap.String("dir", pfx.Dir))
		return err
	}
	defer closeIO()
	if err := thyme.RunArgs(args, ctx, options, nil, runIO); err != nil {
		log.Error("rosemary failed", zap.Error(err), zap.Strings("args", args))
		return err
	}
	return nil
}

// EnsurePrefix creates pfx with rosemary's defaults if it doesn't exist yet
func EnsurePrefix(ctx context.Context, log *zap.Logger, pfx prefix.Prefix) error {
	if pfx.Exists() {
		return nil
	}
	exe, err := etc.GetRosemaryPath()
	if err != nil {
		log.Error("rosemary not found", zap.Error(err))
		return err
	}
	runIO, closeIO, err := rosemaryIO(log, pfx)
	if err != nil {
		log.Error("failed to resolve prefix", zap.Error(err), zap.String("dir", pfx.Dir))
		return err
	}
	defer closeIO()
	log.Info("creating prefix", zap.String("dir", pfx.Dir))
	if err := pfx.Create(ctx, exe, prefix.CreateOptions{}, runIO); err != nil {
		log.Error("failed to create prefix", zap.Error(err), zap.String("dir", pfx.Dir))
		return err
	}
	return nil
}

// RosemaryVersion returns the output of rosemary --version
func RosemaryVersion(ctx context.Context) (string, error) {
	exe, err := etc.GetRosemaryPath()
	if err != nil {
		return "", err
	}
	return etc.RosemaryVersion(ctx, exe, cmd.RosemaryOptions{})
}
