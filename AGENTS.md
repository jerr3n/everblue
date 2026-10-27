# AGENTS.md

Notes for coding agents working on Everblue. See [README.md](README.md) for
what the project does and how users run it.

## Overview

Everblue is a Go launcher that runs the macOS build of Roblox Studio on Linux
through [rosemary](https://github.com/microlift/rosemary), an external tool
that runs Mach-O executables and `.app` bundles. Everblue doesn't run anything
itself: it downloads files, then shells out to `rosemary run <path>`.

## Layout

| File | Purpose |
| --- | --- |
| `main.go` | CLI entry point ([urfave/cli v3](https://github.com/urfave/cli)) and zap logger setup. The bare command runs `runLauncher`. |
| `actions.go` | Core operations shared by the CLI and GUI: `installStudio`, `runStudio`, `installedVersion`, `studioInstalled`, `studioApp`. |
| `setup.go` | Creates `~/.local/share/everblue/versions`. |
| `gui.go` | Qt 6 GUI via [miqt](https://github.com/mappu/miqt): the main window (`buildManager`), the progress dialog, and `runTask`, which runs work off the main thread. |
| `util/fetch.go` | Roblox version lookup (`GetVersion`), download and zip extraction (`DownloadVersion`), and `ProgressFunc`. |
| `util/rosemary.go` | Running rosemary in a prefix (`RunRosemary`), creating the prefix (`EnsurePrefix`) and `RosemaryVersion`, all through [thyme](https://github.com/jerr3n/thyme). |
| `util/util.go` | Small filesystem helpers. |
| `assets/` | The SVG logo (embedded into the binary with `go:embed`) and the `.desktop` entry. |

## Building and checking

```sh
env GOMAXPROCS=4 go build -p 4 -o everblue .
go vet ./...
gofmt -l .
```

- Use `-p 4` / `GOMAXPROCS=4` for every build so it doesn't take all cores.
- The first build compiles the miqt qt6 cgo package and takes more than ten
  minutes. Run it in the background and don't treat the wait as a hang. Later
  builds use the cache.
- The user's shell is fish, so set environment variables with `env VAR=x cmd`,
  not `VAR=x cmd`.
- There are no tests yet. The network code talks to Roblox's CDN and the rest
  needs rosemary and a display, so check changes by building and, when it
  matters, running the binary.

## Behavior to keep

- **The download is the installer, on purpose.** `DownloadVersion` fetches
  `https://setup.rbxcdn.com/mac/<clientVersionUpload>-RobloxStudio.zip`, which
  holds only `RobloxStudioInstaller.app`. `installStudio` then runs that
  installer through rosemary, and the installer puts Studio into the rosemary
  prefix. Don't flag this as a bug or switch to per-package downloads unless
  asked.
- **Default Qt look.** The GUI uses stock Qt widgets with default styling. Don't
  add stylesheets, force a style such as Fusion, set `QT_QPA_PLATFORMTHEME`, or
  build per-desktop looks. Users theme Qt themselves. Use `QIcon_FromTheme`
  for icons, as the existing code does.
- **Zip extraction safety.** `DownloadVersion` rejects entries and symlinks
  that escape the destination directory and recreates `.app` symlinks
  (`Frameworks/*/Versions/Current`). Keep both when changing extraction.

## Conventions

- Paths are built from the `home` directory string that `main` passes down,
  using `fmt.Sprintf("%s/.local/share/...", home)`. Follow that rather than
  looking up the home directory again.
- Log with zap. In `util`, log the error with context fields
  (`logger.Error("...", zap.Error(err), zap.String(...))`) and then return it.
  Messages are short and lowercase.
- Long-running work takes a `context.Context` so the GUI's Cancel button can
  stop it, and reports progress through `util.ProgressFunc` (`total` is `-1`
  when unknown). Callers may pass `nil` for progress.
- Qt is single-threaded. Do slow work in a goroutine and touch widgets only
  inside `mainthread.Start(func() { ... })`. `runTask` and `checkForUpdates`
  show the pattern.
- The GUI keeps a few labels and actions in sync through `refresh()`. Call it
  after anything that changes the installed version or the busy state.
- Comments are sparse and explain why, not what. Match that.
