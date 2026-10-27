package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jerr3n/everblue/util"
	"github.com/jerr3n/thyme/cmd"
	"github.com/jerr3n/thyme/prefix"
	"go.uber.org/zap"
)

// studioPrefix is the rosemary prefix Studio is installed into and run from
func studioPrefix(home string) prefix.Prefix {
	return prefix.Prefix{Dir: fmt.Sprintf("%s/.local/share/rosemary/prefixes/default", home)}
}

func studioApp(home string) string {
	// the prefix's guest / is its root directory
	return filepath.Join(studioPrefix(home).Dir, "root/Applications/RobloxStudio.app")
}

func installStudio(ctx context.Context, home string, logger *zap.Logger, progress util.ProgressFunc) error {
	if progress == nil {
		progress = func(string, int64, int64) {}
	}
	setup(home, logger)
	progress("Checking version", 0, -1)
	data, err := util.GetVersion(ctx, logger)
	if err != nil {
		return err
	}
	if err := util.DownloadVersion(ctx, logger, *data, home, progress); err != nil {
		return err
	}
	progress("Preparing prefix", 0, -1)
	if err := util.EnsurePrefix(ctx, logger, studioPrefix(home)); err != nil {
		return err
	}
	progress("Running installer", 0, -1)
	return util.RunRosemary(ctx, logger, studioPrefix(home), cmd.RosemaryOptions{}, []string{fmt.Sprintf("%s/.local/share/everblue/versions/latest/RobloxStudioInstaller.app", home)})
}

func runStudio(ctx context.Context, home string, logger *zap.Logger) error {
	return util.RunRosemary(ctx, logger, studioPrefix(home), cmd.RosemaryOptions{}, []string{studioApp(home)})
}

// installedVersion returns the version the latest symlink points to, or "" if none
func installedVersion(home string) string {
	target, err := os.Readlink(fmt.Sprintf("%s/.local/share/everblue/versions/latest", home))
	if err != nil {
		return ""
	}
	return filepath.Base(target)
}

func studioInstalled(home string) bool {
	_, err := os.Stat(studioApp(home))
	return err == nil
}
