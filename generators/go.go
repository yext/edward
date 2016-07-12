package generators

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yext/edward/services"
	"github.com/yext/errgo"
)

func init() {
	RegisterGenerator("go", goGenerator)
}

var goGenerator = func(path string) ([]*services.ServiceConfig, []*services.ServiceGroupConfig, error) {
	var outServices []*services.ServiceConfig
	var outGroups []*services.ServiceGroupConfig

	err := validateDir(path)
	if err != nil {
		return outServices, outGroups, err
	}

	visitor := NewGoWalker(filepath.Join(path, "gocode", "src"))
	err = filepath.Walk(filepath.Join(path, "gocode", "src", "yext"), visitor.visit)
	if err != nil {
		return outServices, outGroups, err
	}
	outServices = append(outServices, visitor.GetServices()...)

	return outServices, outGroups, nil
}

type GoWalker struct {
	found  map[string]string
	goPath string
}

func NewGoWalker(goPath string) GoWalker {
	return GoWalker{
		found:  make(map[string]string),
		goPath: goPath,
	}
}

func (v *GoWalker) visit(path string, f os.FileInfo, err error) error {
	if _, err := os.Stat(path); err != nil {
		return errgo.Mask(err)
	}

	if !f.Mode().IsRegular() {
		return nil
	}
	if filepath.Ext(path) != ".go" {
		return nil
	}

	input, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	packageExpr := regexp.MustCompile(`package main\n`)
	if packageExpr.Match(input) {
		packageName := filepath.Base(filepath.Dir(path))
		packagePath := strings.Replace(filepath.Dir(path), v.goPath+"/", "", 1)
		v.found[packageName] = packagePath
	}

	return nil
}

func (v *GoWalker) GetServices() []*services.ServiceConfig {
	var outServices []*services.ServiceConfig

	for packageName, packagePath := range v.found {
		outServices = append(outServices, goService(packageName, packagePath))
	}

	return outServices
}

func goService(name string, goPackage string) *services.ServiceConfig {
	pathStr := "$ALPHA"
	return &services.ServiceConfig{
		Name: name,
		Path: &pathStr,
		Env:  []string{"YEXT_RABBITMQ=localhost"},
		Commands: services.ServiceConfigCommands{
			Build:  "go install " + goPackage,
			Launch: name,
		},
		Properties: services.ServiceConfigProperties{
			Started: "Listening",
		},
	}
}
