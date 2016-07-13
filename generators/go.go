package generators

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/yext/edward/services"
	"github.com/yext/errgo"
)

func init() {
	RegisterGenerator(&GoGenerator{})
}

type GoGenerator struct {
	basePath string
	found    map[string]string
}

func (v *GoGenerator) Name() string {
	return "go"
}

func (v *GoGenerator) StartWalk(path string) {
	v.basePath = path
	v.found = make(map[string]string)
}

func (v *GoGenerator) StopWalk() {
}

func (v *GoGenerator) VisitDir(path string, f os.FileInfo, err error) error {
	if err != nil {
		return errgo.Mask(err)
	}

	if _, err := os.Stat(path); err != nil {
		return errgo.Mask(err)
	}

	files, _ := ioutil.ReadDir(path)
	for _, f := range files {
		fPath := filepath.Join(path, f.Name())
		if filepath.Ext(fPath) != ".go" {
			return nil
		}

		input, err := ioutil.ReadFile(fPath)
		if err != nil {
			return errgo.Mask(err)
		}

		packageExpr := regexp.MustCompile(`package main\n`)
		if packageExpr.Match(input) {
			packageName := filepath.Base(path)
			packagePath, err := filepath.Rel(v.basePath, path)
			if err != nil {
				return errgo.Mask(err)
			}
			v.found[packageName] = packagePath
		}

	}

	return nil
}

func (v *GoGenerator) Found() []*services.ServiceConfig {
	var outServices []*services.ServiceConfig

	for packageName, packagePath := range v.found {
		outServices = append(outServices, goService(packageName, packagePath))
	}

	return outServices
}

func goService(name, packagePath string) *services.ServiceConfig {
	return &services.ServiceConfig{
		Name: name,
		Path: &packagePath,
		Env:  []string{},
		Commands: services.ServiceConfigCommands{
			Build:  "go install",
			Launch: name,
		},
		Properties: services.ServiceConfigProperties{
			Started: "Listening",
		},
	}
}
