package generators

import (
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
)

// EdwardGenerator generates imports for all Edward config files in the directory
// hierarchy.
type EdwardGenerator struct {
	generatorBase
	found []string
}

// Name returns 'edward' to identify this generator.
func (v *EdwardGenerator) Name() string {
	return "edward"
}

// VisitDir searches a directory for edward.json files, and will store an import
// for any found. Returns true in the first return value if an import was found.
func (v *EdwardGenerator) VisitDir(path string) (bool, error) {
	if path == v.basePath {
		return false, nil
	}
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

// Imports returns all imports found during previous walks.
func (v *EdwardGenerator) Imports() []string {
	return v.found
}
