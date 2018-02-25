package generators

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
	"github.com/yext/edward/services"
	"github.com/yext/edward/services/backends/commandline"
)

// IcbmGenerator generates services from an icbm build.spec file
type IcbmGenerator struct {
	generatorBase
	foundServices []*services.ServiceConfig
}

// Name returns 'icbm' to identify this generator
func (v *IcbmGenerator) Name() string {
	return "icbm"
}

// VisitDir checks a directory for a build.spec file. If found, it will parse the file
// to obtain service definitions.
// Once a spec file has been parsed, true, filepath.SkipDir will be returned to ensure
// no further directories below this are parsed.
func (v *IcbmGenerator) VisitDir(path string) (bool, error) {
	buildFilePath := filepath.Join(path, "build.spec")

	if _, err := os.Stat(buildFilePath); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, errors.WithStack(err)
	}

	specData, err := ioutil.ReadFile(buildFilePath)
	if err != nil {
		return false, errors.WithStack(err)
	}

	relPath, err := filepath.Rel(v.basePath, path)
	if err != nil {
		return false, errors.WithStack(err)
	}

	v.foundServices = append(v.foundServices, parsePlayServices(relPath, specData)...)
	v.foundServices = append(v.foundServices, parseJavaServices(relPath, specData)...)

	return true, filepath.SkipDir
}

// Services returns a slice of all the services generated during this walk
func (v *IcbmGenerator) Services() []*services.ServiceConfig {
	return v.foundServices
}

func parsePlayServices(path string, spec []byte) []*services.ServiceConfig {
	var outServices []*services.ServiceConfig

	playExpr := regexp.MustCompile("name=\"(.*)_dev")
	matches := playExpr.FindAllSubmatch(spec, -1)

	for _, match := range matches {
		if len(match) > 1 {
			outServices = append(outServices, playService(path, string(match[1])))
		}
	}

	return outServices
}

func parseJavaServices(path string, spec []byte) []*services.ServiceConfig {
	var outServices []*services.ServiceConfig

	playExpr := regexp.MustCompile("name=\"([A-Za-z0-9]+)\"")
	matches := playExpr.FindAllSubmatch(spec, -1)

	for _, match := range matches {
		if len(match) > 1 {
			outServices = append(outServices, javaService(path, string(match[1])))
		}
	}

	return outServices
}

func playService(path, name string) *services.ServiceConfig {
	return &services.ServiceConfig{
		Name: name,
		Path: &path,
		Env:  []string{},
		BackendConfig: &commandline.CommandLineBackend{
			Commands: commandline.ServiceConfigCommands{
				Build:  "python tools/icbm/build.py :" + name + "_dev",
				Launch: "thirdparty/play/play test src/com/yext/" + name,
			},
		},
	}
}

func javaService(path, name string) *services.ServiceConfig {
	return &services.ServiceConfig{
		Name: name,
		Path: &path,
		Env:  []string{},
		BackendConfig: &commandline.CommandLineBackend{
			Commands: commandline.ServiceConfigCommands{
				Build:  "python tools/icbm/build.py :" + name,
				Launch: "build/" + name + "/" + name,
			},
		},
	}
}
