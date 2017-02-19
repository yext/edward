package generators

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

type GoGenerator struct {
	generatorBase
	found map[string]string
}

func (v *GoGenerator) Name() string {
	return "go"
}

func (v *GoGenerator) StartWalk(path string) {
	v.generatorBase.StartWalk(path)
	v.found = make(map[string]string)
}

func (v *GoGenerator) StopWalk() {
}

func (v *GoGenerator) VisitDir(path string) (bool, error) {
	files, _ := ioutil.ReadDir(path)
	for _, f := range files {
		fPath := filepath.Join(path, f.Name())
		if filepath.Ext(fPath) != ".go" {
			continue
		}

		input, err := ioutil.ReadFile(fPath)
		if err != nil {
			return false, errors.WithStack(err)
		}

		packageExpr := regexp.MustCompile(`package main\n`)
		if packageExpr.Match(input) {
			packageName := filepath.Base(path)
			packagePath, err := filepath.Rel(v.basePath, path)
			if err != nil {
				return false, errors.WithStack(err)
			}
			v.found[packageName] = packagePath
			return true, nil
		}

	}

	return false, nil
}

func (v *GoGenerator) Services() []*services.ServiceConfig {
	var outServices []*services.ServiceConfig

	for packageName, packagePath := range v.found {
		service, err := v.goService(packageName, packagePath)
		if err == nil {
			outServices = append(outServices, service)
		}
		// TODO: Log any error?
	}

	return outServices
}

func (v *GoGenerator) goService(name, packagePath string) (*services.ServiceConfig, error) {
	service := &services.ServiceConfig{
		Name: name,
		Path: &packagePath,
		Env:  []string{},
		Commands: services.ServiceConfigCommands{
			Build:  "go install",
			Launch: name,
		},
	}

	watch, err := v.createWatch(service)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	service.SetWatch(watch)

	return service, nil
}

func (v *GoGenerator) createWatch(service *services.ServiceConfig) (services.ServiceWatch, error) {
	return services.ServiceWatch{
		Service:       service,
		IncludedPaths: v.getImportList(service),
	}, nil
}

func (v *GoGenerator) getImportList(service *services.ServiceConfig) []string {
	if service.Path == nil {
		return nil
	}

	// Get a list of imports using 'go list'
	var imports = []string{}
	cmd := exec.Command("go", "list", "-f", "{{ join .Imports \":\" }}")
	cmd.Dir = *service.Path
	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Println(errBuf.String())
		return []string{*service.Path}
	}
	imports = append(imports, strings.Split(out.String(), ":")...)

	// Verify the import paths exist
	var checkedImports = []string{*service.Path}
	for _, i := range imports {
		path := os.ExpandEnv(fmt.Sprintf("$GOPATH/src/%v", i))
		if _, err := os.Stat(path); err == nil {
			rel, err := filepath.Rel(v.basePath, path)
			if err != nil {
				// TODO: Handle this error more effectively
				fmt.Println(err)
				continue
			}
			checkedImports = append(checkedImports, rel)
		}
	}
	// Remove subpaths
	sort.Strings(checkedImports)
	var outImports []string
	for i, path := range checkedImports {
		include := true
		for j, earlier := range checkedImports {
			if i > j && strings.HasPrefix(path, earlier) {
				include = false
			}
		}
		if include {
			outImports = append(outImports, path)
		}
	}
	return outImports
}
