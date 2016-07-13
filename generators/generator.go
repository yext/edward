package generators

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sabhiram/go-git-ignore"
	"github.com/yext/edward/services"
	"github.com/yext/errgo"
)

type Generator interface {
	Name() string
	StartWalk(basePath string)
	StopWalk()
	VisitDir(path string, f os.FileInfo, err error) error
	Found() []*services.ServiceConfig
}

type ConfigGenerator func(path string) ([]*services.ServiceConfig, []*services.ServiceGroupConfig, error)

var Generators map[string]Generator

func RegisterGenerator(g Generator) {
	if Generators == nil {
		Generators = make(map[string]Generator)
	}
	Generators[g.Name()] = g
}

func loadIgnores(path string, currentIgnores *ignore.GitIgnore) (*ignore.GitIgnore, error) {
	ignoreFile := filepath.Join(path, ".edwardignore")
	if _, err := os.Stat(ignoreFile); err != nil {
		if os.IsNotExist(err) {
			return currentIgnores, nil
		}
		return currentIgnores, errgo.Mask(err)
	}

	ignores, err := ignore.CompileIgnoreFile(ignoreFile)
	return ignores, errgo.Mask(err)
}

func shouldIgnore(basePath, path string, ignores *ignore.GitIgnore) bool {
	rel, err := filepath.Rel(basePath, path)
	if err != nil {
		fmt.Println("Error checking ignore:", err)
		return false
	}

	return ignores.MatchesPath(rel)
}

func GenerateServices(path string) ([]*services.ServiceConfig, error) {
	var outServices []*services.ServiceConfig

	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		if err != nil {
			return outServices, err
		}
		return outServices, errors.New(path + " is not a directory")
	}

	// TODO: Create a stack of ignore files to handle ignores in subdirs
	ignores, err := loadIgnores(path, nil)
	if err != nil {
		return outServices, errgo.Mask(err)
	}

	for name, generator := range Generators {
		generator.StartWalk(path)
		err := filepath.Walk(path, func(curPath string, f os.FileInfo, err error) error {
			if _, err := os.Stat(curPath); err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return errgo.Mask(err)
			}

			if !f.Mode().IsDir() || shouldIgnore(path, curPath, ignores) {
				return nil
			}

			err = generator.VisitDir(curPath, f, err)
			if err == filepath.SkipDir {
				return err
			}
			return errgo.Mask(err)
		})
		generator.StopWalk()
		if err != nil {
			fmt.Println("Error in generator", name, ":", err)
		} else {
			outServices = append(outServices, generator.Found()...)
		}
	}

	return outServices, nil
}
