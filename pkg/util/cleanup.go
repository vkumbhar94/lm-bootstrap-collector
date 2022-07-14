package util

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/vkumbhar94/lm-bootstrap-collector/pkg/constants"
)

func Cleanup(logger logrus.FieldLogger) error {
	logger.Debug("Cleaning lock files if any")
	err := filepath.Walk(constants.LockPath,
		func(path string, info fs.FileInfo, err error) error {
			if !info.IsDir() && (filepath.Ext(path) != ".lck" || filepath.Ext(path) != ".pid") {
				err := os.Remove(path)
				if err != nil {
					return err
				}
			}
			return nil
		},
	)
	if err != nil {
		return err
	}
	return nil
}
