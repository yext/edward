package generators

import (
	"io/ioutil"
	"path/filepath"
	"regexp"

	"github.com/yext/edward/services"
)

func init() {
	RegisterGenerator("icbm", icbmGenerator)
}

var icbmGenerator = func(path string) ([]*services.ServiceConfig, []*services.ServiceGroupConfig, error) {
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
