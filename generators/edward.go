package generators

import (
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
)

type EdwardGenerator struct {
	generatorBase
	found []string
}

func (v *EdwardGenerator) Name() string {
	return "edward"
}

func (v *EdwardGenerator) StopWalk() {
}

func (v *EdwardGenerator) VisitDir(path string) (bool, error) {
	files, _ := ioutil.ReadDir(path)
	for _, f := range files {
		if f.Name() == "edward.json" {
			relPath, err := filepath.Rel(v.basePath, filepath.Join(path, f.Name()))
			if err != nil {
				return false, errors.WithStack(err)
			}
			v.found = append(v.found, relPath)
			return true, nil
		}
	}

	return false, nil
}

func (v *EdwardGenerator) Imports() []string {
	return v.found
}
