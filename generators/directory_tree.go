package generators

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	ignore "github.com/sabhiram/go-git-ignore"
)

// directory represents a directory for the purposes of scanning for projects
// to import.
type directory struct {
	Path     string
	Parent   *directory
	children []*directory
	ignores  *ignore.GitIgnore
}

// newDirectory builds a directory structure under the specified path
func newDirectory(path string, parent *directory) (*directory, error) {
	if parent != nil && parent.Ignores() != nil && parent.Ignores().MatchesPath(path) {
		return nil, nil
	}

	ignores, err := loadIgnores(path, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	d := &directory{
		Path:    path,
		Parent:  parent,
		ignores: ignores,
	}

	for _, file := range files {

		// Follow symlinked dirs
		if file.Mode()&os.ModeSymlink != 0 {
			linkPath, err := os.Readlink(filepath.Join(path, file.Name()))
			if err != nil {
				return nil, errors.WithStack(err)
			}
			resolved := filepath.Join(path, linkPath)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			file, err = os.Stat(resolved)
			if err != nil {
				return nil, errors.WithStack(err)
			}
		}

		if file.IsDir() {
			child, err := newDirectory(filepath.Join(path, file.Name()), d)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			d.children = append(d.children, child)
		}
	}

	return d, nil
}

// Ignores returns the .edwardignore config for this directory or any of its
// ancestor directories.
func (d *directory) Ignores() *ignore.GitIgnore {
	if d.ignores != nil {
		return d.ignores
	}

	if d.Parent != nil {
		return d.Parent.Ignores()
	}
	return nil
}

// SkipAll indicates that all generators should skip this directory and
// subdirectories.
var SkipAll skipAllErr = skipAllErr("skipped")

type skipAllErr string

func (s skipAllErr) Error() string {
	return string(s)
}

func (d *directory) Generate(generators []Generator) error {
	if d == nil || len(generators) == 0 {
		return nil
	}

	var childGenerators []Generator
	for _, generator := range generators {
		_, err := generator.VisitDir(d.Path)
		if err == SkipAll {
			return nil
		}
		if err != nil && err != filepath.SkipDir {
			return errors.WithStack(err)
		}
		if err != filepath.SkipDir {
			childGenerators = append(childGenerators, generator)
		}
	}

	for _, child := range d.children {
		err := child.Generate(childGenerators)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func loadIgnores(path string, currentIgnores *ignore.GitIgnore) (*ignore.GitIgnore, error) {
	ignoreFile := filepath.Join(path, ".edwardignore")
	if _, err := os.Stat(ignoreFile); err != nil {
		if os.IsNotExist(err) {
			return currentIgnores, nil
		}
		return currentIgnores, errors.WithStack(err)
	}

	ignores, err := ignore.CompileIgnoreFile(ignoreFile)
	return ignores, errors.WithStack(err)
}
