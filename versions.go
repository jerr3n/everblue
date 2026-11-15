package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jerr3n/everblue/util"
	"go.uber.org/zap"
)

// The active Studio lives where the installer puts it, Applications/RobloxStudio.app in the prefix. Other
// installed versions wait in a hidden folder next to it, so switching is a rename rather than a copy.

type knownVersion struct {
	Version   string    `json:"version"`
	Channel   string    `json:"channel"`
	Installed time.Time `json:"installed"`
}

type versionState struct {
	Active string                  `json:"active"`
	Known  map[string]knownVersion `json:"known"`
}

// installedStudio is one version of Studio in the prefix
type installedStudio struct {
	Hash    string
	Version string
	Channel string
	Active  bool
}

var hashPattern = regexp.MustCompile(`^version-[0-9a-f]{16}$`)

func versionStatePath(home string) string {
	return fmt.Sprintf("%s/.local/share/everblue/versions.json", home)
}

func versionStore(home string) string {
	return filepath.Join(studioPrefix(home).Dir, "root/Applications/.everblue-versions")
}

func loadVersionState(home string) versionState {
	state := versionState{Known: map[string]knownVersion{}}
	data, err := os.ReadFile(versionStatePath(home))
	if err == nil {
		json.Unmarshal(data, &state)
	}
	if state.Known == nil {
		state.Known = map[string]knownVersion{}
	}
	// Studio installed before Everblue kept this file: the installer's download says which it is
	if state.Active == "" {
		if target, err := os.Readlink(fmt.Sprintf("%s/.local/share/everblue/versions/latest", home)); err == nil {
			state.Active = filepath.Base(target)
		}
	}
	return state
}

func (s versionState) save(home string, logger *zap.Logger) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	path := versionStatePath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		logger.Error("failed to make data dir", zap.Error(err), zap.String("path", path))
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		logger.Error("failed to save versions", zap.Error(err), zap.String("path", path))
		return err
	}
	return nil
}

var bundleVersion = regexp.MustCompile(`<key>CFBundleShortVersionString</key>\s*<string>([^<]*)</string>`)

// appVersion reads the version number out of an .app's Info.plist
func appVersion(app string) string {
	data, err := os.ReadFile(filepath.Join(app, "Contents/Info.plist"))
	if err != nil {
		return ""
	}
	if m := bundleVersion.FindSubmatch(data); m != nil {
		return string(m[1])
	}
	return ""
}

// listVersions is every Studio in the prefix, the active one first and then newest first
func listVersions(home string) []installedStudio {
	state := loadVersionState(home)
	var out []installedStudio
	if studioInstalled(home) && state.Active != "" {
		out = append(out, installedStudio{
			Hash: state.Active, Version: appVersion(studioApp(home)), Channel: state.Known[state.Active].Channel, Active: true,
		})
	}
	entries, _ := os.ReadDir(versionStore(home))
	var stored []installedStudio
	for _, e := range entries {
		app := filepath.Join(versionStore(home), e.Name(), "RobloxStudio.app")
		if _, err := os.Stat(app); err != nil || e.Name() == state.Active {
			continue
		}
		stored = append(stored, installedStudio{Hash: e.Name(), Version: appVersion(app), Channel: state.Known[e.Name()].Channel})
	}
	sort.Slice(stored, func(i, j int) bool { return stored[i].Version > stored[j].Version })
	return append(out, stored...)
}

// shelve moves the active Studio into the store, keeping it as the version state says it is
func shelve(home string, state versionState) error {
	if !studioInstalled(home) {
		return nil
	}
	name := state.Active
	if name == "" {
		name = "unknown-" + time.Now().Format("20060102-150405")
	}
	dir := filepath.Join(versionStore(home), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.Rename(studioApp(home), filepath.Join(dir, "RobloxStudio.app"))
}

// activateVersion makes an installed version the one that runs
func activateVersion(home string, logger *zap.Logger, hash string) error {
	state := loadVersionState(home)
	if hash == state.Active && studioInstalled(home) {
		return nil
	}
	app := filepath.Join(versionStore(home), hash, "RobloxStudio.app")
	if _, err := os.Stat(app); err != nil {
		logger.Error("version not installed", zap.String("version", hash))
		return fmt.Errorf("%s isn't installed", hash)
	}
	if err := shelve(home, state); err != nil {
		logger.Error("failed to put away the active studio", zap.Error(err))
		return err
	}
	if err := os.Rename(app, studioApp(home)); err != nil {
		logger.Error("failed to activate studio", zap.Error(err), zap.String("version", hash))
		return err
	}
	os.Remove(filepath.Dir(app))
	state.Active = hash
	logger.Info("activated studio", zap.String("version", hash))
	return state.save(home, logger)
}

// deleteVersion removes a version that isn't active
func deleteVersion(home string, logger *zap.Logger, hash string) error {
	if hash == loadVersionState(home).Active {
		return errors.New("the active version can't be deleted, switch to another first")
	}
	dir := filepath.Join(versionStore(home), hash)
	if filepath.Dir(dir) != versionStore(home) {
		return fmt.Errorf("bad version %q", hash)
	}
	logger.Info("deleting studio", zap.String("version", hash))
	return os.RemoveAll(dir)
}

// installVersion downloads a whole Studio of a channel and makes it the active one
func installVersion(ctx context.Context, home string, logger *zap.Logger, channel, hash string, progress util.ProgressFunc) error {
	if progress == nil {
		progress = func(string, int64, int64) {}
	}
	if !hashPattern.MatchString(hash) {
		return fmt.Errorf("%q isn't a Studio version, they look like version-0123456789abcdef", hash)
	}
	setup(home, logger)
	if err := util.EnsureStudioRosemary(ctx, logger, progress); err != nil {
		return err
	}
	progress("Preparing prefix", 0, -1)
	if err := util.EnsurePrefix(ctx, logger, studioPrefix(home)); err != nil {
		return err
	}
	zipPath := fmt.Sprintf("%s/.local/share/everblue/downloads/%s.zip", home, hash)
	defer os.Remove(zipPath)
	if err := util.DownloadFile(ctx, logger, util.AppZipURL(channel, hash), zipPath, progress); err != nil {
		return err
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		logger.Error("failed to read zip", zap.Error(err))
		return err
	}
	defer zr.Close()
	dest := filepath.Join(versionStore(home), hash)
	// a half-extracted version from a cancelled install is no use
	os.RemoveAll(dest)
	if err := util.ExtractZip(ctx, logger, &zr.Reader, dest, progress); err != nil {
		os.RemoveAll(dest)
		return err
	}
	state := loadVersionState(home)
	state.Known[hash] = knownVersion{Version: appVersion(filepath.Join(dest, "RobloxStudio.app")), Channel: channel, Installed: time.Now()}
	if err := state.save(home, logger); err != nil {
		return err
	}
	progress("Switching versions", 0, -1)
	return activateVersion(home, logger, hash)
}

// recordInstalled notes the version the installer put in place
func recordInstalled(home string, logger *zap.Logger, hash, channel string) {
	state := loadVersionState(home)
	state.Active = hash
	state.Known[hash] = knownVersion{Version: appVersion(studioApp(home)), Channel: channel, Installed: time.Now()}
	state.save(home, logger)
}

// shortHash is enough of a version hash to tell versions apart
func shortHash(hash string) string {
	return strings.TrimPrefix(hash, "version-")
}
