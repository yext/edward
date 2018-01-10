package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
)

func GetConfigPathFromWorkingDirectory(homeDir string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return GetConfigPath(homeDir, wd), nil
}

// GetConfigPath identifies the location of edward.json, if any exists
func GetConfigPath(homeDir string, wd string) string {
	var pathOptions []string

	// Config file in Edward Config dir
	pathOptions = append(pathOptions, filepath.Join(homeDir, "edward.json"))

	// Config file in current working directory
	pathOptions = append(pathOptions, filepath.Join(wd, "edward.json"))
	for path.Dir(wd) != wd {
		wd = path.Dir(wd)
		pathOptions = append(pathOptions, filepath.Join(wd, "edward.json"))
	}

	for _, path := range pathOptions {
		_, err := os.Stat(path)
		if err != nil {
			continue
		}
		absfp, absErr := filepath.Abs(path)
		if absErr != nil {
			fmt.Println("Error getting config file: ", absErr)
			return ""
		}
		return absfp
	}

	return ""
}
