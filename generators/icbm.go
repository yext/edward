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
	RegisterGenerator(&IcbmGenerator{})
}

type IcbmGenerator struct {
	foundServices []*services.ServiceConfig
}

func (v *IcbmGenerator) Name() string {
	return "icbm"
}

func (v *IcbmGenerator) StartWalk(path string) {
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
	v.foundServices = append(v.foundServices, parsePlayServices(specData)...)
	v.foundServices = append(v.foundServices, parseJavaServices(specData)...)

	return filepath.SkipDir
}

func (v *IcbmGenerator) Found() []*services.ServiceConfig {
	return v.foundServices
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
