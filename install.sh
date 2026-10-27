#!/usr/bin/env bash
# Downloads the latest Everblue release from GitHub and installs it, its desktop entry and its
# icon for the current user.
#
#   ./install.sh              install Qt runtime dependencies, download, install
#   ./install.sh --no-deps    skip the package manager step
#   ./install.sh --uninstall  remove what this script installed
#
# VERSION picks a release tag instead of the latest one.
set -euo pipefail

repo="jerr3n/everblue"
bin_dir="$HOME/.local/bin"
app_dir="$HOME/.local/share/applications"
icon_dir="$HOME/.local/share/icons/hicolor/scalable/apps"

info() { printf '\033[1;34m::\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mwarning:\033[0m %s\n' "$*" >&2; }
die() { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

refresh_caches() {
	command -v update-desktop-database >/dev/null && update-desktop-database -q "$app_dir" || true
	command -v gtk-update-icon-cache >/dev/null && gtk-update-icon-cache -q -t "$HOME/.local/share/icons/hicolor" || true
}

uninstall() {
	info "removing everblue"
	rm -f "$bin_dir/everblue" "$app_dir/everblue.desktop" "$icon_dir/everblue.svg"
	refresh_caches
	info "done; downloaded installers in ~/.local/share/everblue and the rosemary prefix were left alone"
}

install_deps() {
	local id=""
	if [[ -r /etc/os-release ]]; then
		# shellcheck disable=SC1091
		id="$(. /etc/os-release && echo "$ID ${ID_LIKE:-}")"
	fi

	# qt6 svg is needed too: without its image plugin the logo renders blank
	case " $id " in
	*" arch "*)
		info "installing dependencies with pacman"
		sudo pacman -S --needed qt6-base qt6-svg
		;;
	*" fedora "*)
		info "installing dependencies with dnf"
		sudo dnf install qt6-qtbase-gui qt6-qtsvg
		;;
	*" debian "* | *" ubuntu "*)
		info "installing dependencies with apt"
		sudo apt install libqt6widgets6 qt6-qpa-plugins qt6-svg-plugins
		;;
	*)
		warn "unknown distro; install the Qt 6 base libraries and the Qt 6 svg plugin yourself"
		;;
	esac
}

deps=1
case "${1:-}" in
"") ;;
--no-deps) deps=0 ;;
--uninstall) uninstall; exit 0 ;;
-h | --help) sed -n '2,10s/^# \{0,1\}//p' "$0"; exit 0 ;;
*) die "unknown option: $1 (see --help)" ;;
esac

[[ $EUID -ne 0 ]] || die "run this as your normal user; it asks for sudo when it needs it"
[[ "$(uname -m)" == x86_64 ]] || die "releases are only built for x86_64; build from source instead (see README.md)"
command -v curl >/dev/null || die "curl not found"

((deps)) && install_deps

tag="${VERSION:-}"
if [[ -z "$tag" ]]; then
	# github redirects releases/latest to releases/tag/<tag>, which avoids the rate-limited API
	tag="$(curl -fsSI "https://github.com/$repo/releases/latest" | sed -n 's|^location: .*/tag/\([^[:space:]]*\).*|\1|Ip')"
	[[ -n "$tag" ]] || die "could not find the latest release of $repo"
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

info "downloading everblue $tag"
curl -fL --progress-bar -o "$tmp/everblue" "https://github.com/$repo/releases/download/$tag/everblue"
curl -fsSL -o "$tmp/everblue.desktop" "https://raw.githubusercontent.com/$repo/$tag/assets/everblue.desktop"
curl -fsSL -o "$tmp/everblue.svg" "https://raw.githubusercontent.com/$repo/$tag/assets/everblue.svg"

info "installing to $bin_dir"
install -Dm755 "$tmp/everblue" "$bin_dir/everblue"
install -Dm644 "$tmp/everblue.desktop" "$app_dir/everblue.desktop"
install -Dm644 "$tmp/everblue.svg" "$icon_dir/everblue.svg"
refresh_caches

if missing="$(ldd "$bin_dir/everblue" 2>/dev/null | grep 'not found')"; then
	warn "everblue is missing shared libraries; install Qt 6 (see --help for --no-deps):"
	printf '%s\n' "$missing" >&2
fi
case ":$PATH:" in
*":$bin_dir:"*) ;;
*) warn "$bin_dir is not on your PATH; add it so the desktop entry and the everblue command work" ;;
esac
command -v rosemary >/dev/null || warn "rosemary not found on PATH; install it from https://github.com/microlift/rosemary"

info "done; run 'everblue' or open Everblue from your application menu"
