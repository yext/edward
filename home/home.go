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

func createDirIfNeeded(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0777)
	}
}

func NewConfiguration() (*EdwardConfiguration, error) {
	cfg := &EdwardConfiguration{}
	err := cfg.Initialize()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return cfg, nil
}

// Initialize sets up EdwardConfig based on the location of .edward in the home dir
func (e *EdwardConfiguration) Initialize() error {
	user, err := user.Current()
	if err != nil {
		return errors.WithStack(err)
	}
	return e.InitializeWithDir(path.Join(user.HomeDir, ".edward"))
}

func (e *EdwardConfiguration) InitializeWithDir(dir string) error {
	e.Dir = dir
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
