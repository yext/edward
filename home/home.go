package home

import (
	"os"
	"os/user"
	"path"

	"github.com/pkg/errors"
)

type EdwardConfiguration struct {
	Dir          string
	EdwardLogDir string
	LogDir       string
	PidDir       string
	ScriptDir    string
}

var EdwardConfig EdwardConfiguration = EdwardConfiguration{}

func createDirIfNeeded(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0777)
	}
}

func (e *EdwardConfiguration) Initialize() error {
	user, err := user.Current()
	if err != nil {
		return errors.WithStack(err)
	}
	e.Dir = path.Join(user.HomeDir, ".edward")
	createDirIfNeeded(e.Dir)
	e.EdwardLogDir = path.Join(e.Dir, "edward_logs")
	createDirIfNeeded(e.EdwardLogDir)
	e.LogDir = path.Join(e.Dir, "logs")
	createDirIfNeeded(e.LogDir)
	e.PidDir = path.Join(e.Dir, "pidFiles")
	createDirIfNeeded(e.PidDir)
	e.ScriptDir = path.Join(e.Dir, "scriptFiles")
	createDirIfNeeded(e.ScriptDir)
	return nil
}
