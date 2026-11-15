package util

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

// RunRosemary runs a Mach-O executable or .app bundle and its arguments in pfx, with env added to ours
func RunRosemary(ctx context.Context, log *zap.Logger, pfx prefix.Prefix, options cmd.RosemaryOptions, env []string, args []string) error {
	runIO, closeIO, err := rosemaryIO(log, pfx)
	if err != nil {
		log.Error("failed to resolve prefix", zap.Error(err), zap.String("dir", pfx.Dir))
		return err
	}
	defer closeIO()
	runIO.Env = append(runIO.Env, env...)
	exe, err := RosemaryPath()
	if err != nil {
		log.Error("rosemary not found", zap.Error(err))
		return err
	}
	if err := thyme.RunArgs(args, ctx, options, &exe, runIO); err != nil {
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
	exe, err := RosemaryPath()
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

// InstallBase installs the Darwin base into an existing pfx that doesn't have one.
// rosemary prefix create keeps an existing prefix's files and only adds what's missing.
func InstallBase(ctx context.Context, log *zap.Logger, pfx prefix.Prefix) error {
	exe, err := RosemaryPath()
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
	log.Info("installing base", zap.String("dir", pfx.Dir))
	if err := pfx.Create(ctx, exe, prefix.CreateOptions{}, runIO); err != nil {
		log.Error("failed to install base", zap.Error(err), zap.String("dir", pfx.Dir))
		return err
	}
	return nil
}

// RosemaryVersion returns the output of rosemary --version
func RosemaryVersion(ctx context.Context) (string, error) {
	exe, err := RosemaryPath()
	if err != nil {
		return "", err
	}
	return etc.RosemaryVersion(ctx, exe, cmd.RosemaryOptions{})
}

// Studio needs the Roblox Studio fork of rosemary, which upstream rosemary can't stand in for. Everblue
// keeps its own copy of it rather than replacing whatever rosemary the system has.

// StudioRosemaryRelease is where the fork's newest build is published
const StudioRosemaryRelease = "https://github.com/3XPLwastaken/rosemary-studio/releases/latest/download/"

const studioRosemaryAsset = "rosemary-studio-x86_64-linux.tar.gz"

// studioVariant is what the fork's rosemary --version says that upstream's doesn't
const studioVariant = "variant: studio"

var managedRosemary string

// UseManagedRosemary tells the util package where Everblue keeps its own rosemary
func UseManagedRosemary(home string) {
	managedRosemary = fmt.Sprintf("%s/.local/share/everblue/rosemary/rosemary", home)
}

// RosemaryPath is the rosemary Everblue runs: its own copy once it has one, else the one on PATH
func RosemaryPath() (string, error) {
	if managedRosemary != "" {
		if info, err := os.Stat(managedRosemary); err == nil && !info.IsDir() {
			return managedRosemary, nil
		}
	}
	return exec.LookPath("rosemary")
}

// IsStudioRosemary reports whether exe is the Roblox Studio build of rosemary
func IsStudioRosemary(ctx context.Context, exe string) bool {
	version, err := etc.RosemaryVersion(ctx, exe, cmd.RosemaryOptions{})
	return err == nil && strings.Contains(version, studioVariant)
}

// RosemaryState says which rosemary Everblue would run, and whether it's the Studio build
func RosemaryState(ctx context.Context) (path string, studio bool) {
	path, err := RosemaryPath()
	if err != nil {
		return "", false
	}
	return path, IsStudioRosemary(ctx, path)
}

// EnsureStudioRosemary makes sure the rosemary Everblue runs is the Studio build, downloading it into
// Everblue's own folder when it's missing or the one found is upstream's
func EnsureStudioRosemary(ctx context.Context, log *zap.Logger, progress ProgressFunc) error {
	if path, studio := RosemaryState(ctx); studio {
		return nil
	} else if path != "" {
		log.Info("rosemary isn't the studio build, installing it", zap.String("found", path))
	} else {
		log.Info("rosemary not found, installing the studio build")
	}
	if managedRosemary == "" {
		return errors.New("no folder to install rosemary into")
	}
	if progress == nil {
		progress = func(string, int64, int64) {}
	}
	dir := filepath.Dir(managedRosemary)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Error("failed to make rosemary dir", zap.Error(err), zap.String("dir", dir))
		return err
	}

	progress("Checking rosemary", 0, -1)
	sums, err := get(ctx, StudioRosemaryRelease+"SHA256SUMS")
	if err != nil {
		log.Error("failed to get rosemary checksums", zap.Error(err))
		return err
	}
	want := ""
	scanner := bufio.NewScanner(sums.Body)
	for scanner.Scan() {
		if fields := strings.Fields(scanner.Text()); len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == studioRosemaryAsset {
			want = fields[0]
		}
	}
	sums.Body.Close()
	if want == "" {
		log.Error("no checksum for rosemary", zap.String("asset", studioRosemaryAsset))
		return fmt.Errorf("the rosemary release has no checksum for %s", studioRosemaryAsset)
	}

	archive := filepath.Join(dir, studioRosemaryAsset)
	defer os.Remove(archive)
	if err := DownloadFile(ctx, log, StudioRosemaryRelease+studioRosemaryAsset, archive, func(_ string, done, total int64) {
		progress("Downloading rosemary", done, total)
	}); err != nil {
		return err
	}
	if got, err := fileSHA256(archive); err != nil || got != want {
		log.Error("rosemary download doesn't match its checksum", zap.Error(err), zap.String("want", want), zap.String("got", got))
		return errors.New("the rosemary download doesn't match its checksum")
	}

	progress("Installing rosemary", 0, -1)
	staged := managedRosemary + ".new"
	if err := extractRosemary(archive, staged); err != nil {
		os.Remove(staged)
		log.Error("failed to unpack rosemary", zap.Error(err))
		return err
	}
	if !IsStudioRosemary(ctx, staged) {
		os.Remove(staged)
		log.Error("downloaded rosemary isn't the studio build")
		return errors.New("the downloaded rosemary doesn't run, or isn't the Studio build")
	}
	if err := os.Rename(staged, managedRosemary); err != nil {
		log.Error("failed to install rosemary", zap.Error(err))
		return err
	}
	log.Info("installed studio rosemary", zap.String("path", managedRosemary))
	return nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// extractRosemary copies the rosemary binary, and only it, out of the release archive
func extractRosemary(archive, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return errors.New("the archive has no rosemary in it")
		}
		if err != nil {
			return err
		}
		if header.Typeflag != tar.TypeReg || filepath.Base(header.Name) != "rosemary" {
			continue
		}
		out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	}
}
