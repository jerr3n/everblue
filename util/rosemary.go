package util

import (
	"context"
	"os/exec"
	"reflect"
	"slices"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapio"
)

type RosemaryOptions struct {
	Slow        bool `rosemary:"slow"`
	Native      bool `rosemary:"native"`
	Emulated    bool `rosemary:"emulated"`
	ObjcDbg     bool `rosemary:"objc-debug"`
	NewInstance bool `rosemary:"new"`
}

// RosemaryFlags I don't use this. Someone might need to some day.
func RosemaryFlags(options RosemaryOptions) []string {
	flags := []string{}

	v := reflect.ValueOf(options)
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		name, ok := t.Field(i).Tag.Lookup("rosemary")
		if !ok || name == "" {
			continue
		}
		if v.Field(i).Kind() == reflect.Bool && v.Field(i).Bool() {
			flags = append(flags, "--"+name)
		}
	}

	return flags
}

func RunRosemary(ctx context.Context, log *zap.Logger, options RosemaryOptions, xc []string) error {
	stdout := &zapio.Writer{Log: log.Named("rosemary"), Level: zap.InfoLevel}
	defer stdout.Close()

	stderr := &zapio.Writer{Log: log.Named("rosemary"), Level: zap.ErrorLevel}
	defer stderr.Close()

	flags := RosemaryFlags(options)

	cmd := exec.CommandContext(ctx, "rosemary", slices.Concat(flags, xc)...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// RosemaryVersion returns the output of rosemary --version
func RosemaryVersion(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "rosemary", "--version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
