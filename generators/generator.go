package generators

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yext/edward/services"
	"github.com/yext/errgo"
)

type ConfigGenerator func(path string) ([]*services.ServiceConfig, []*services.ServiceGroupConfig, error)

var Generators map[string]ConfigGenerator = map[string]ConfigGenerator{
	"icbm": func(path string) ([]*services.ServiceConfig, []*services.ServiceGroupConfig, error) {
		var outServices []*services.ServiceConfig
		var outGroups []*services.ServiceGroupConfig

		err := validateDir(path)
		if err != nil {
			return outServices, outGroups, err
		}

		buildFilePath := filepath.Join(path, "build.spec")
		err = validateRegular(buildFilePath)
		if err != nil {
			return outServices, outGroups, err
		}

		specData, err := ioutil.ReadFile(buildFilePath)
		if err != nil {
			return outServices, outGroups, err
		}
		outServices = append(outServices, parsePlayServices(specData)...)
		outServices = append(outServices, parseJavaServices(specData)...)

		return outServices, outGroups, nil
	},
	"go": func(path string) ([]*services.ServiceConfig, []*services.ServiceGroupConfig, error) {
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
	},
}

func validateRegular(path string) error {
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		if err != nil {
			return err
		}
		return errors.New(path + " is not a regular file")
	}
	return nil
}

func parsePlayServices(spec []byte) []*services.ServiceConfig {
	var outServices []*services.ServiceConfig

	playExpr := regexp.MustCompile("name=\"(.*)_dev")
	matches := playExpr.FindAllSubmatch(spec, -1)

	for _, match := range matches {
		if len(match) > 1 {
			outServices = append(outServices, playService(string(match[1])))
		}
	}

	return outServices
}

func parseJavaServices(spec []byte) []*services.ServiceConfig {
	var outServices []*services.ServiceConfig

	playExpr := regexp.MustCompile("name=\"([A-Za-z0-9]+)\"")
	matches := playExpr.FindAllSubmatch(spec, -1)

	for _, match := range matches {
		if len(match) > 1 {
			outServices = append(outServices, javaService(string(match[1])))
		}
	}

	return outServices
}

func validateDir(path string) error {
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		if err != nil {
			return err
		}
		return errors.New(path + " is not a directory")
	}
	return nil
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

func GenerateServices(path string) ([]*services.ServiceConfig, []*services.ServiceGroupConfig, error) {

	var outServices []*services.ServiceConfig
	var outGroups []*services.ServiceGroupConfig

	err := validateDir(path)
	if err != nil {
		return outServices, outGroups, err
	}

	for name, generator := range Generators {
		s, g, err := generator(path)
		if err != nil {
			fmt.Println("Error in generator", name, ":", err)
		} else {
			outServices = append(outServices, s...)
			outGroups = append(outGroups, g...)
		}
	}

	return outServices, outGroups, nil
}

func playService(name string) *services.ServiceConfig {
	pathStr := "$ALPHA"
	return &services.ServiceConfig{
		Name: name,
		Path: &pathStr,
		Env:  []string{},
		Commands: services.ServiceConfigCommands{
			Build:  "python tools/icbm/build.py :" + name + "_dev",
			Launch: "thirdparty/play/play test src/com/yext/" + name,
		},
		Properties: services.ServiceConfigProperties{
			Started: "Server is up and running",
		},
	}
}

func javaService(name string) *services.ServiceConfig {
	pathStr := "$ALPHA"
	return &services.ServiceConfig{
		Name: name,
		Path: &pathStr,
		Env:  []string{},
		Commands: services.ServiceConfigCommands{
			Build:  "python tools/icbm/build.py :" + name,
			Launch: "JVM_ARGS='-Xmx3G' build/" + name + "/" + name,
		},
		Properties: services.ServiceConfigProperties{
			Started: "Listening",
		},
	}
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
