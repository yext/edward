package home

import (
	"os"
	"os/user"
	"path"
)

type EdwardConfiguration struct {
	Dir       string
	LogDir    string
	PidDir    string
	ScriptDir string
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
		return err
	}
	e.Dir = path.Join(user.HomeDir, ".edward")
	e.LogDir = path.Join(e.Dir, "logs")
	e.PidDir = path.Join(e.Dir, "pidFiles")
	e.ScriptDir = path.Join(e.Dir, "scriptFiles")
	createDirIfNeeded(e.Dir)
	createDirIfNeeded(e.LogDir)
	createDirIfNeeded(e.PidDir)
	createDirIfNeeded(e.ScriptDir)
	return nil
}
