package util

import (
	"fmt"
	"os"
	"path/filepath"
)

func NewestDir(parent string) (string, error) {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return "", err
	}

	var newest string
	var newestTime int64

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if t := info.ModTime().UnixNano(); newest == "" || t > newestTime {
			newest, newestTime = e.Name(), t
		}
	}

	if newest == "" {
		return "", fmt.Errorf("no directories found in %s", parent)
	}
	return filepath.Join(parent, newest), nil
}
