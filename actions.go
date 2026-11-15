package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
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

func installStudio(ctx context.Context, home string, logger *zap.Logger, display displaySystem, progress util.ProgressFunc) error {
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
	// the installer launches Studio when it's done, so it gets the same display system
	return util.RunRosemary(ctx, logger, studioPrefix(home), cmd.RosemaryOptions{}, display.env(), []string{fmt.Sprintf("%s/.local/share/everblue/versions/latest/RobloxStudioInstaller.app", home)})
}

func runStudio(ctx context.Context, home string, logger *zap.Logger, display displaySystem) error {
	logger.Info("launching studio", zap.Stringer("display", display))
	return util.RunRosemary(ctx, logger, studioPrefix(home), cmd.RosemaryOptions{}, display.env(), []string{studioApp(home)})
}

// installedVersion returns the version the latest symlink points to, or "" if none
func installedVersion(home string) string {
	target, err := os.Readlink(fmt.Sprintf("%s/.local/share/everblue/versions/latest", home))
	if err != nil {
		return ""
	}
	return filepath.Base(target)
}

// check is the result of one diagnostic
type check struct {
	Name   string
	OK     bool
	Detail string
	// fixable checks can be repaired from Everblue, the rest need the user
	Fixable bool
}

// prefixHasBase reports whether the Studio prefix has a Darwin base installed.
// Without one the installer can't find /usr/bin/unzip and stops.
func prefixHasBase(home string) (bool, error) {
	pfx := studioPrefix(home)
	if err := pfx.ResolveYAML(); err != nil {
		return false, err
	}
	return pfx.State.Base.Project != "", nil
}

func diagnose(ctx context.Context, home string) []check {
	base := check{Name: "Prefix base system"}
	switch ok, err := prefixHasBase(home); {
	case errors.Is(err, fs.ErrNotExist):
		base.OK = true
		base.Detail = "The prefix doesn't exist yet. Installing Studio creates it with a base system."
	case err != nil:
		base.Detail = fmt.Sprintf("Couldn't read the prefix's state: %v", err)
	case ok:
		base.OK = true
		base.Detail = "The prefix has a Darwin base system installed."
	default:
		base.Fixable = true
		base.Detail = "The prefix has no Darwin base system, so it's missing macOS tools such as /usr/bin/unzip. " +
			"The Studio installer can't unpack Studio without them."
	}

	webkit := check{Name: "WebKitGTK"}
	if path := util.FindLibrary(ctx, util.WebKitGTKSoname); path != "" {
		webkit.OK = true
		webkit.Detail = "Found " + path + "."
	} else {
		webkit.Detail = util.WebKitGTKSoname + " isn't installed, so Studio's login window stays blank. " +
			"Install it with your package manager: webkitgtk-6.0 on Arch, webkitgtk6.0 on Fedora, " +
			"libwebkitgtk-6.0-4 on Debian and Ubuntu."
	}
	return []check{base, webkit}
}

func installBase(ctx context.Context, home string, logger *zap.Logger) error {
	return util.InstallBase(ctx, logger, studioPrefix(home))
}

func studioInstalled(home string) bool {
	_, err := os.Stat(studioApp(home))
	return err == nil
}
