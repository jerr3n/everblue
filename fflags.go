package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// Fast flags live in Everblue's config rather than in Studio, which an update or a version switch replaces.
// They're written into the active Studio every time it launches.

type fastFlag struct {
	Name  string
	Value any
}

func fflagsPath(home string) string {
	return fmt.Sprintf("%s/.config/everblue/fflags.json", home)
}

func loadFastFlags(home string) []fastFlag {
	data, err := os.ReadFile(fflagsPath(home))
	if err != nil {
		return nil
	}
	return parseFastFlags(data)
}

// parseFastFlags reads a ClientAppSettings.json style object, sorted by name
func parseFastFlags(data []byte) []fastFlag {
	var raw map[string]any
	if json.Unmarshal(data, &raw) != nil {
		return nil
	}
	flags := make([]fastFlag, 0, len(raw))
	for name, value := range raw {
		flags = append(flags, fastFlag{name, value})
	}
	sort.Slice(flags, func(i, j int) bool { return flags[i].Name < flags[j].Name })
	return flags
}

func fastFlagsJSON(flags []fastFlag) []byte {
	raw := make(map[string]any, len(flags))
	for _, f := range flags {
		if strings.TrimSpace(f.Name) != "" {
			raw[strings.TrimSpace(f.Name)] = f.Value
		}
	}
	data, _ := json.MarshalIndent(raw, "", "  ")
	return data
}

func saveFastFlags(home string, logger *zap.Logger, flags []fastFlag) error {
	path := fflagsPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		logger.Error("failed to make config dir", zap.Error(err), zap.String("path", path))
		return err
	}
	if err := os.WriteFile(path, fastFlagsJSON(flags), 0o644); err != nil {
		logger.Error("failed to save fast flags", zap.Error(err), zap.String("path", path))
		return err
	}
	return nil
}

// flagValue types what was typed the way Studio expects: true/false and numbers aren't strings
func flagValue(text string) any {
	text = strings.TrimSpace(text)
	switch strings.ToLower(text) {
	case "true":
		return true
	case "false":
		return false
	}
	if n, err := strconv.ParseInt(text, 10, 64); err == nil {
		return n
	}
	return text
}

func flagText(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprint(v)
	}
}

// applyFastFlags writes the saved flags where Studio reads them
func applyFastFlags(home string, logger *zap.Logger) error {
	if !studioInstalled(home) {
		return nil
	}
	flags := loadFastFlags(home)
	dir := filepath.Join(studioApp(home), "Contents/MacOS/ClientSettings")
	path := filepath.Join(dir, "ClientAppSettings.json")
	if len(flags) == 0 {
		// leave nothing behind from flags that were removed
		os.Remove(path)
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		logger.Error("failed to make client settings dir", zap.Error(err), zap.String("path", dir))
		return err
	}
	if err := os.WriteFile(path, fastFlagsJSON(flags), 0o644); err != nil {
		logger.Error("failed to write fast flags", zap.Error(err), zap.String("path", path))
		return err
	}
	logger.Info("applied fast flags", zap.Int("count", len(flags)))
	return nil
}
