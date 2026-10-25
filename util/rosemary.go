package util

import (
	"os/exec"
	"reflect"
	"slices"

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

func RunRosemary(log *zap.Logger, options RosemaryOptions, xc []string) error {
	stdout := &zapio.Writer{Log: log.Named("rosemary"), Level: zap.InfoLevel}
	defer stdout.Close()

	stderr := &zapio.Writer{Log: log.Named("rosemary"), Level: zap.ErrorLevel}
	defer stderr.Close()

	flags := RosemaryFlags(options)

	cmd := exec.Command("rosemary", slices.Concat(flags, xc)...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
