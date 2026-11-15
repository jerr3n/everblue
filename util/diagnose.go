package util

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WebKitGTKSoname is the library rosemary loads to give macOS web views content
const WebKitGTKSoname = "libwebkitgtk-6.0.so.4"

// libraryDirs are searched when ldconfig isn't available
var libraryDirs = []string{"/usr/lib", "/usr/lib64", "/usr/lib/x86_64-linux-gnu", "/lib", "/lib64", "/lib/x86_64-linux-gnu"}

// FindLibrary returns where the dynamic linker would find soname, or "" if it wouldn't
func FindLibrary(ctx context.Context, soname string) string {
	// LD_LIBRARY_PATH wins over the cache for rosemary too
	for _, dir := range filepath.SplitList(os.Getenv("LD_LIBRARY_PATH")) {
		if path := filepath.Join(dir, soname); dir != "" && fileExists(path) {
			return path
		}
	}
	if out, err := ldconfigCache(ctx); err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(out))
		for scanner.Scan() {
			// "\tlibfoo.so.1 (libc6,x86-64) => /usr/lib/libfoo.so.1"
			name, rest, ok := strings.Cut(strings.TrimSpace(scanner.Text()), " ")
			if !ok || name != soname || !strings.Contains(rest, "x86-64") {
				continue
			}
			if _, path, ok := strings.Cut(rest, "=> "); ok {
				return path
			}
		}
		return ""
	}
	for _, dir := range libraryDirs {
		if path := filepath.Join(dir, soname); fileExists(path) {
			return path
		}
	}
	return ""
}

func ldconfigCache(ctx context.Context) ([]byte, error) {
	// Debian keeps ldconfig in /sbin, which often isn't on a user's PATH
	exe, err := exec.LookPath("ldconfig")
	if err != nil {
		exe = "/sbin/ldconfig"
	}
	return exec.CommandContext(ctx, exe, "-p").Output()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
