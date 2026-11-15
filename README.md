# Everblue

Everblue runs Roblox Studio on Linux. It downloads the macOS build of Studio
from Roblox, runs its installer through
[rosemary](https://github.com/microlift/rosemary), and then launches Studio
from the rosemary prefix. Its Qt 6 window installs, updates and launches
Studio, keeps several Studio versions and release channels, and manages fast
flags; a CLI does the basics from a terminal.

## Requirements

- **The Roblox Studio build of rosemary**, from
  [rosemary-studio](https://github.com/3XPLwastaken/rosemary-studio), an
  experimental fork of [rosemary](https://github.com/microlift/rosemary) with
  the fixes Studio needs. rosemary is what actually runs the macOS `.app`
  bundles. You don't need to install it yourself: if the rosemary on your
  `PATH` isn't the Studio build (its `rosemary --version` says
  `variant: studio`), or there's none, Everblue downloads the Studio build into
  `~/.local/share/everblue/rosemary/` before installing or launching Studio and
  uses that copy, leaving any other rosemary alone. Everblue uses rosemary's
  default prefix at `~/.local/share/rosemary/prefixes/default`.
- **Go 1.27** or newer.
- **Qt 6** development files, a C++ compiler and `pkg-config`, for the
  [miqt](https://github.com/mappu/miqt) bindings. On Fedora:

  ```sh
  sudo dnf install qt6-qtbase-devel gcc-c++ pkgconf-pkg-config
  ```

  On Debian/Ubuntu:

  ```sh
  sudo apt install qt6-base-dev build-essential pkg-config
  ```

## Quick install

`install.sh` installs the Qt 6 runtime libraries (Arch, Fedora and
Debian/Ubuntu), downloads the latest x86_64 release from GitHub and does the
steps under [Installing](#installing) for you. You don't need Go or a clone of
the repo:

```sh
curl -fsSL https://raw.githubusercontent.com/jerr3n/everblue/main/install.sh | bash
```

From a clone, run `./install.sh`. Pass `--no-deps` to skip the package manager,
`--uninstall` to remove Everblue, and set `VERSION=<tag>` to install a specific
release.

## Building

```sh
go build -o everblue .
```

The first build compiles the miqt Qt bindings through cgo and takes a long
time (more than ten minutes is normal). Later builds use the Go build cache
and are fast. To keep the build from using every core, limit it:

```sh
env GOMAXPROCS=4 go build -p 4 -o everblue .
```

## Installing

Put the binary somewhere on your `PATH`, then install the desktop entry and
icon so Everblue shows up in your application menu:

```sh
install -Dm755 everblue ~/.local/bin/everblue
install -Dm644 assets/everblue.desktop ~/.local/share/applications/everblue.desktop
install -Dm644 assets/everblue.svg ~/.local/share/icons/hicolor/scalable/apps/everblue.svg
```

## Usage

| Command | What it does |
| --- | --- |
| `everblue` | Launch Studio. If Studio isn't installed yet, open the Everblue window instead. Exits when Studio exits. |
| `everblue cfg` (`config`) | Open the Everblue window. |
| `everblue install` | Download the latest Studio and run its installer. |
| `everblue run` | Run Studio from the terminal, with no window. |
| `everblue setup` (`init`, `initialize`) | Create Everblue's data directories. `install` does this for you. |

The Everblue window shows the installed and latest Studio versions, the
rosemary version and prefix, and buttons to install, update or run Studio.
**View → Show Log** (Ctrl+L) opens a log pane with everything Everblue and
rosemary print.

### First run

1. Run `everblue` (or open it from your application menu).
2. Studio isn't installed yet, so the Everblue window opens. Click
   **Install Studio**.
3. Everblue downloads the installer and starts it. Finish the installer's
   own steps in its window.
4. Click **Run Studio**. From then on, `everblue` launches Studio directly.

To update, open the window with `everblue cfg` and click **Update Studio**
when a newer version is available.

## Where things live

| Path | Contents |
| --- | --- |
| `~/.local/share/everblue/versions/<version>/` | Downloaded Studio installers, one directory per version. |
| `~/.local/share/everblue/versions/latest` | Symlink to the most recently downloaded version. |
| `~/.local/share/rosemary/prefixes/default/root/Applications/RobloxStudio.app` | The installed Studio, which is what Everblue launches. |

## Theming

Everblue uses stock Qt widgets with Qt's default styling and doesn't ship a
theme of its own. If it doesn't match your desktop, theme Qt the usual way,
for example with Kvantum (and a theme such as KvLibadwaita on GNOME) or
`qt6ct`.

## License

Everblue is licensed under the GNU General Public License v3.0. See
[LICENSE](LICENSE).
