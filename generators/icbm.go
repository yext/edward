package generators

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/juju/errgo"
	"github.com/yext/edward/services"
)

func init() {
	RegisterGenerator(&IcbmGenerator{})
}

type IcbmGenerator struct {
	basePath      string
	foundServices []*services.ServiceConfig
}

func (v *IcbmGenerator) Name() string {
	return "icbm"
}

func (v *IcbmGenerator) StartWalk(path string) {
	v.basePath = path
	v.foundServices = nil
}

func (v *IcbmGenerator) StopWalk() {
}

func (v *IcbmGenerator) VisitDir(path string, f os.FileInfo, err error) error {
	buildFilePath := filepath.Join(path, "build.spec")

	if _, err := os.Stat(buildFilePath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errgo.Mask(err)
	}

	specData, err := ioutil.ReadFile(buildFilePath)
	if err != nil {
		return errgo.Mask(err)
	}

	relPath, err := filepath.Rel(v.basePath, path)
	if err != nil {
		return err
	}

	v.foundServices = append(v.foundServices, parsePlayServices(relPath, specData)...)
	v.foundServices = append(v.foundServices, parseJavaServices(relPath, specData)...)

	return filepath.SkipDir
}

func (v *IcbmGenerator) Found() []*services.ServiceConfig {
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
		Commands: services.ServiceConfigCommands{
			Build:  "python tools/icbm/build.py :" + name + "_dev",
			Launch: "thirdparty/play/play test src/com/yext/" + name,
		},
	}
}

func javaService(path, name string) *services.ServiceConfig {
	return &services.ServiceConfig{
		Name: name,
		Path: &path,
		Env:  []string{},
		Commands: services.ServiceConfigCommands{
			Build:  "python tools/icbm/build.py :" + name,
			Launch: "build/" + name + "/" + name,
		},
	}
}
