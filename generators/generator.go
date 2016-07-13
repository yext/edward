package generators

import (
	"fmt"
	"os"
	"path/filepath"

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

func GenerateServices(path string) ([]*services.ServiceConfig, error) {
	var outServices []*services.ServiceConfig

	err := validateDir(path)
	if err != nil {
		return outServices, err
	}

	for name, generator := range Generators {
		generator.StartWalk(path)
		err := filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
			if _, err := os.Stat(path); err != nil {
				return errgo.Mask(err)
			}

			if f.Mode().IsDir() {
				err := generator.VisitDir(path, f, err)
				if err == filepath.SkipDir {
					return err
				}
				return errgo.Mask(err)
			}
			return nil
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
