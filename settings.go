package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// displaySystem is how rosemary shows Studio's windows
type displaySystem string

const (
	// displayAsk asks before every launch
	displayAsk displaySystem = ""
	// displayWayland uses native Wayland windows. On Hyprland, rosemary also reads window positions over
	// Hyprland's IPC, since Wayland never tells a program where its windows are and Qt needs that to dock panels.
	displayWayland displaySystem = "wayland"
	// displayXWayland runs Studio as an X11 program, where positions are known on every desktop
	displayXWayland displaySystem = "xwayland"
)

type settings struct {
	Display displaySystem `json:"display,omitempty"`
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

// onHyprland reports whether we're running inside a Hyprland session
func onHyprland() bool {
	return os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != ""
}

// env is what rosemary needs set to use d. Without a choice, Wayland is what rosemary picks anyway.
func (d displaySystem) env() []string {
	if d == displayXWayland {
		return []string{"SDL_VIDEO_DRIVER=x11"}
	}
	return []string{"SDL_VIDEO_DRIVER=wayland", "ROSEMARY_HYPRLAND_IPC=1"}
}

func (d displaySystem) String() string {
	switch d {
	case displayWayland:
		return "Wayland"
	case displayXWayland:
		return "XWayland"
	}
	return "Ask every launch"
}
