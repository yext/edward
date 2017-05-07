package home

import (
	"os"
	"os/user"
	"path"

	"github.com/pkg/errors"
)

// EdwardConfiguration defines the application config for Edward
type EdwardConfiguration struct {
	Dir          string
	EdwardLogDir string
	LogDir       string
	PidDir       string
	StateDir     string
	ScriptDir    string
}

// EdwardConfig stores a shared instance of EdwardConfiguration for use across the app
var EdwardConfig = EdwardConfiguration{}

func createDirIfNeeded(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0777)
	}
}

// Initialize sets up EdwardConfig based on the location of .edward in the home dir
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
	e.StateDir = path.Join(e.Dir, "stateFiles")
	createDirIfNeeded(e.StateDir)
	e.ScriptDir = path.Join(e.Dir, "scriptFiles")
	createDirIfNeeded(e.ScriptDir)
	return nil
}
