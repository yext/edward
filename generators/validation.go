package generators

import (
	"errors"
	"os"
)

func validateRegular(path string) error {
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		if err != nil {
			return err
		}
		return errors.New(path + " is not a regular file")
	}
	return nil
}

func validateDir(path string) error {
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		if err != nil {
			return err
		}
		return errors.New(path + " is not a directory")
	}
	return nil
}
