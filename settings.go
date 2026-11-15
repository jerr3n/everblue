package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jerr3n/everblue/util"
	"go.uber.org/zap"
)

// displaySystem is how rosemary shows Studio's windows
type displaySystem string

const (
	// displayAsk asks before every launch
	displayAsk displaySystem = ""
	// displayWayland uses native Wayland windows. On Hyprland and KDE Plasma, rosemary also reads and sets
	// window positions through the compositor, since Wayland never tells a program where its windows are and
	// Qt needs that to dock panels.
	displayWayland displaySystem = "wayland"
	// displayXWayland runs Studio as an X11 program, where positions are known on every desktop
	displayXWayland displaySystem = "xwayland"
)

type settings struct {
	Display displaySystem `json:"display,omitempty"`
	// Accent is the UI's highlight color, accentBlue or accentRed
	Accent string `json:"accent,omitempty"`
	// Channel is the release channel updates come from, LIVE when empty
	Channel string `json:"channel,omitempty"`
	// StillLogo stops the logo spinning
	StillLogo bool `json:"stillLogo,omitempty"`
	// Theme is Everblue's own colors (themeEverblue), the desktop's colors in Everblue's look (themeColors),
	// or the desktop's own Qt style and colors (themePlain)
	Theme string `json:"theme,omitempty"`
	// SystemTheme is the old switch for themePlain, only read so that choice carries over
	SystemTheme bool `json:"systemTheme,omitempty"`
	// ShowLogo puts the spinning logo in the home page's bottom right corner
	ShowLogo bool `json:"showLogo,omitempty"`
	// Classic is the older sidebar window instead of the compact single column one
	Classic bool `json:"classic,omitempty"`
}

const (
	themeEverblue = ""
	themeColors   = "colors"
	themePlain    = "plain"
)

// channel is the release channel to follow
func (s settings) channel() string {
	if s.Channel == "" {
		return util.LiveChannel
	}
	return s.Channel
}

func settingsPath(home string) string {
	return fmt.Sprintf("%s/.config/everblue/settings.json", home)
}

// loadSettings returns the saved settings, or the defaults when there are none or they can't be read
func loadSettings(home string, logger *zap.Logger) settings {
	var s settings
	data, err := os.ReadFile(settingsPath(home))
	if errors.Is(err, fs.ErrNotExist) {
		return s
	}
	if err == nil {
		err = json.Unmarshal(data, &s)
	}
	if err != nil {
		logger.Error("failed to read settings, using defaults", zap.Error(err), zap.String("path", settingsPath(home)))
		return settings{}
	}
	if s.SystemTheme {
		s.Theme, s.SystemTheme = themePlain, false
	}
	return s
}

func (s settings) save(home string, logger *zap.Logger) error {
	path := settingsPath(home)
	data, err := json.MarshalIndent(s, "", "  ")
	if err == nil {
		err = os.MkdirAll(filepath.Dir(path), 0o755)
	}
	if err == nil {
		err = os.WriteFile(path, append(data, '\n'), 0o644)
	}
	if err != nil {
		logger.Error("failed to save settings", zap.Error(err), zap.String("path", path))
	}
	return err
}

// compositorDesktop names the desktop rosemary can ask where windows are, or "" if it can't
func compositorDesktop() string {
	if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != "" {
		return "Hyprland"
	}
	for _, desktop := range strings.Split(os.Getenv("XDG_CURRENT_DESKTOP"), ":") {
		if desktop == "KDE" {
			return "KDE Plasma"
		}
	}
	return ""
}

// env is what rosemary needs set to use d. Without a choice, Wayland is what rosemary picks anyway.
func (d displaySystem) env() []string {
	if d == displayXWayland {
		return []string{"SDL_VIDEO_DRIVER=x11"}
	}
	// rosemary uses whichever compositor it finds, and nothing on any other
	return []string{"SDL_VIDEO_DRIVER=wayland", "ROSEMARY_COMPOSITOR_IPC=1"}
}

func (d displaySystem) String() string {
	switch d {
	case displayWayland:
		return "Wayland (recommended)"
	case displayXWayland:
		return "XWayland"
	}
	return "Ask every launch"
}
