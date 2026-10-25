package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jerr3n/everblue/util"
	"go.uber.org/zap"
)

func studioApp(home string) string {
	// ~/.local/share/rosemary/prefixes/default/
	return fmt.Sprintf("%s/.local/share/rosemary/prefixes/default/root/Applications/RobloxStudio.app", home)
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
	progress("Running installer", 0, -1)
	return util.RunRosemary(ctx, logger, util.RosemaryOptions{}, []string{"run", fmt.Sprintf("%s/.local/share/everblue/versions/latest/RobloxStudioInstaller.app", home)})
}

func runStudio(ctx context.Context, home string, logger *zap.Logger) error {
	return util.RunRosemary(ctx, logger, util.RosemaryOptions{}, []string{"run", studioApp(home)})
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
