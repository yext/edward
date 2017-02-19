package generators

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func init() {
	RegisterGenerator(&EdwardGenerator{})
}

type EdwardGenerator struct {
	generatorBase
	found []string
}

func (v *EdwardGenerator) Name() string {
	return "edward"
}

func (v *EdwardGenerator) StopWalk() {
}

func (v *EdwardGenerator) VisitDir(path string, f os.FileInfo, err error) error {
	if err != nil {
		return errors.WithStack(err)
	}

	if _, err := os.Stat(path); err != nil {
		return errors.WithStack(err)
	}

	files, _ := ioutil.ReadDir(path)
	for _, f := range files {
		if f.Name() == "edward.json" {
			relPath, err := filepath.Rel(v.basePath, filepath.Join(path, f.Name()))
			if err != nil {
				return errors.WithStack(err)
			}
			v.found = append(v.found, relPath)
		}
	}

	return nil
}

func (v *EdwardGenerator) Imports() []string {
	return v.found
}
