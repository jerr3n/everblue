package util

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

type APIVersion struct {
	Version             string `json:"version"`
	ClientVersionUpload string `json:"clientVersionUpload"`
	BootstrapperVersion string `json:"bootstrapperVersion"`
}

func GetVersion(logger *zap.Logger) (*APIVersion, error) {
	resp, err := http.Get("https://clientsettingscdn.roblox.com/v2/client-version/MacStudio")
	if err != nil {
		logger.Error("failed to get current roblox studio version", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("error during reading data", zap.Error(err))
		return nil, err
	}
	var data APIVersion
	if err := json.Unmarshal(body, &data); err != nil {
		logger.Error("error unmarshalling json", zap.Error(err))
		return nil, err
	}
	return &data, nil
}

func DownloadVersion(logger *zap.Logger, version APIVersion, home string) error {
	// use https://setup.rbxcdn.com/mac/<version>-RobloxStudio.zip, clientversionupload is <version>
	url := fmt.Sprintf("https://setup.rbxcdn.com/mac/%s-RobloxStudio.zip", version.ClientVersionUpload)
	resp, err := http.Get(url)
	if err != nil {
		logger.Error("failed to download current roblox studio version", zap.Error(err))
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.Error("unexpected status downloading roblox studio", zap.Int("status", resp.StatusCode), zap.String("url", url))
		return fmt.Errorf("unexpected status %d downloading %s", resp.StatusCode, url)
	}
	dest := fmt.Sprintf("%s/.local/share/everblue/versions/%s", home, version.ClientVersionUpload)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		err := os.Mkdir(dest, 0755)
		if err != nil {
			logger.Error("error creating version dir", zap.Error(err), zap.String("version", version.ClientVersionUpload))
			return err
		}
	} else if err != nil {
		logger.Error("unknown error", zap.Error(err))
		return err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("unknown error", zap.Error(err))
		return err
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		logger.Error("failed to read zip", zap.Error(err))
		return err
	}

	for _, f := range zr.File {
		target := filepath.Join(dest, f.Name)
		if rel, err := filepath.Rel(dest, target); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			logger.Error("illegal path", zap.String("path", target))
			return fmt.Errorf("illegal path in zip: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				logger.Error("failed to make dir", zap.Error(err))
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			logger.Error("failed to make dir", zap.Error(err))
			return err
		}
		rc, err := f.Open()
		if err != nil {
			logger.Error("failed to open zip entry", zap.Error(err), zap.String("path", target))
			return err
		}

		// .app bundles contain symlinks (Frameworks/*/Versions/Current), the link target is stored as the file contents
		if f.Mode()&os.ModeSymlink != 0 {
			link, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				logger.Error("failed to read symlink", zap.Error(err), zap.String("path", target))
				return err
			}
			linkTarget := string(link)
			rel, err := filepath.Rel(dest, filepath.Join(filepath.Dir(target), linkTarget))
			if filepath.IsAbs(linkTarget) || err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
				logger.Error("illegal symlink", zap.String("path", target), zap.String("link", linkTarget))
				return fmt.Errorf("illegal symlink %s -> %s", f.Name, linkTarget)
			}
			if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
				logger.Error("failed to remove existing file", zap.Error(err), zap.String("path", target))
				return err
			}
			if err := os.Symlink(linkTarget, target); err != nil {
				logger.Error("failed to create symlink", zap.Error(err), zap.String("path", target))
				return err
			}
			continue
		}

		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode().Perm())
		if err != nil {
			rc.Close()
			logger.Error("failed to create file", zap.Error(err), zap.String("path", target))
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		if err != nil {
			out.Close()
			logger.Error("failed to extract file", zap.Error(err), zap.String("path", target))
			return err
		}
		if err := out.Close(); err != nil {
			logger.Error("failed to close file", zap.Error(err), zap.String("path", target))
			return err
		}
	}
	latest := fmt.Sprintf("%s/.local/share/everblue/versions/latest", home)
	if err := os.Remove(latest); err != nil && !os.IsNotExist(err) {
		logger.Error("error removing old latest link", zap.Error(err))
		return err
	}
	err = os.Symlink(dest, latest)
	if err != nil {
		logger.Error("error in symlinking", zap.Error(err))
	}
	return nil
}
