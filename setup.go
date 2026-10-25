package main

import (
	"fmt"
	"os"

	"go.uber.org/zap"
)

func setup(home string, logger *zap.Logger) {
	if _, err := os.Stat(fmt.Sprintf("%s/.local/share/everblue", home)); os.IsNotExist(err) {
		logger.Warn("everblue share directory not existent yet")
		err := os.Mkdir(fmt.Sprintf("%s/.local/share/everblue", home), 0755)
		if err != nil {
			logger.Fatal("error creating everblue dir", zap.Error(err))
		}
		err = os.Mkdir(fmt.Sprintf("%s/.local/share/everblue/versions", home), 0755)
		if err != nil {
			logger.Fatal("error creating versions dir", zap.Error(err))
		}
	} else if err != nil {
		logger.Fatal("unknown error", zap.Error(err))
	} else {
		logger.Debug("directory exists")
	}
}
