package main

import (
	"bufio"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func generateConfigFile(file string) error {

	serviceList := []ServiceConfig{}
	for _, val := range services {
		serviceList = append(serviceList, *val)
	}

	groupList := []ServiceGroupConfig{}
	for _, val := range groups {
		groupList = append(groupList, *val)
	}

	cfg := NewConfig(serviceList, groupList)

	f, err := os.Create(file)
	if err != nil {
		return err
	}

	defer f.Close()

	w := bufio.NewWriter(f)
	err = cfg.Save(w)
	if err != nil {
		return err
	}

	w.Flush()

	return nil
}

func validateRegular(path string) error {
	if info, err := os.Stat(path); !info.Mode().IsRegular() {
		if err != nil {
			return err
		}
		return errors.New(path + " is not a regular file")
	}
	return nil
}

func validateDir(path string) error {
	if info, err := os.Stat(path); !info.IsDir() {
		if err != nil {
			return err
		}
		return errors.New(path + " is not a directory")
	}
	return nil
}

type ConfigGenerator func(path string) ([]*ServiceConfig, []*ServiceGroupConfig, error)

func parsePlayServices(spec []byte) []*ServiceConfig {
	var outServices []*ServiceConfig

	playExpr := regexp.MustCompile("name=\"(.*)_dev")
	matches := playExpr.FindAllSubmatch(spec, -1)

	for _, match := range matches {
		if len(match) > 1 {
			outServices = append(outServices, playService(string(match[1])))
		}
	}

	return outServices
}

func parseJavaServices(spec []byte) []*ServiceConfig {
	var outServices []*ServiceConfig

	playExpr := regexp.MustCompile("name=\"([A-Za-z0-9]+)\"")
	matches := playExpr.FindAllSubmatch(spec, -1)

	for _, match := range matches {
		if len(match) > 1 {
			outServices = append(outServices, javaService(string(match[1])))
		}
	}

	return outServices
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

func (v *GoWalker) GetServices() []*ServiceConfig {
	var outServices []*ServiceConfig

	for packageName, packagePath := range v.found {
		outServices = append(outServices, goService(packageName, packagePath))
	}

	return outServices
}

var Generators map[string]ConfigGenerator = map[string]ConfigGenerator{
	"icbm": func(path string) ([]*ServiceConfig, []*ServiceGroupConfig, error) {
		var outServices []*ServiceConfig
		var outGroups []*ServiceGroupConfig

		err := validateDir(path)
		if err != nil {
			return outServices, outGroups, err
		}

		// TODO: Look for build.spec and parse for Play services vs regular java ones
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
	"go": func(path string) ([]*ServiceConfig, []*ServiceGroupConfig, error) {
		var outServices []*ServiceConfig
		var outGroups []*ServiceGroupConfig

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

func generateServices(path string) ([]*ServiceConfig, []*ServiceGroupConfig, error) {

	var outServices []*ServiceConfig
	var outGroups []*ServiceGroupConfig

	err := validateDir(path)
	if err != nil {
		return outServices, outGroups, err
	}

	for _, generator := range Generators {
		s, g, err := generator(path)
		if err != nil {
			log.Print(err)
		} else {
			outServices = append(outServices, s...)
			outGroups = append(outGroups, g...)
		}
	}

	return outServices, outGroups, nil
}

func playService(name string) *ServiceConfig {
	pathStr := "$ALPHA"
	return &ServiceConfig{
		Name: name,
		Path: &pathStr,
		Env:  []string{"YEXT_RABBITMQ=localhost"},
		Commands: ServiceConfigCommands{
			Build:  "python tools/icbm/build.py :" + name + "_dev",
			Launch: "thirdparty/play/play test src/com/yext/" + name,
		},
		Properties: ServiceConfigProperties{
			Started: "Server is up and running",
		},
	}
}

func javaService(name string) *ServiceConfig {
	pathStr := "$ALPHA"
	return &ServiceConfig{
		Name: name,
		Path: &pathStr,
		Env:  []string{"YEXT_RABBITMQ=localhost", "YEXT_SITE=office"},
		Commands: ServiceConfigCommands{
			Build:  "python tools/icbm/build.py :" + name,
			Launch: "JVM_ARGS='-Xmx3G' build/" + name + "/" + name,
		},
		Properties: ServiceConfigProperties{
			Started: "Listening",
		},
	}
}

func goService(name string, goPackage string) *ServiceConfig {
	pathStr := "$ALPHA"
	return &ServiceConfig{
		Name: name,
		Path: &pathStr,
		Env:  []string{"YEXT_RABBITMQ=localhost"},
		Commands: ServiceConfigCommands{
			Build:  "go install " + goPackage,
			Launch: name,
		},
		Properties: ServiceConfigProperties{
			Started: "Listening",
		},
	}
}

func applyHardCodedServicesAndGroups() {
	services["rabbitmq"] = thirdPartyService("rabbitmq", "rabbitmq-server", "rabbitmqctl stop", "completed")
	// TODO: haproxy actually needs a kill -9 to effectively die
	// TODO: haproxy also doesn't have an effective start output
	services["haproxy"] = thirdPartyService("haproxy", "sudo $ALPHA/tools/bin/haproxy_localhost.sh", "", "backend")

	groups["thirdparty"] = &ServiceGroupConfig{
		Name: "thirdparty",
		Services: []*ServiceConfig{
			services["rabbitmq"],
			services["haproxy"],
		},
	}

	groups["stormgrp"] = &ServiceGroupConfig{
		Name: "stormgrp",
		Groups: []*ServiceGroupConfig{
			groups["thirdparty"],
		},
		Services: []*ServiceConfig{
			services["admin2"],
			services["users"],
			services["storm"],
			services["locationsstorm"],
			services["ProfileServer"],
		},
	}

	groups["pages"] = &ServiceGroupConfig{
		Name: "pages",
		Groups: []*ServiceGroupConfig{
			groups["stormgrp"],
		},
		Services: []*ServiceConfig{
			services["sites-staging"],
			services["sites-storm"],
			services["sites-cog"],
		},
	}

	groups["resellers"] = &ServiceGroupConfig{
		Name: "resellers",
		Groups: []*ServiceGroupConfig{
			groups["storm"],
		},
		Services: []*ServiceConfig{
			services["resellersapi"],
			services["subscriptions"],
			services["SalesApiServer"],
		},
	}

	groups["bag"] = &ServiceGroupConfig{
		Name: "bag",
		Groups: []*ServiceGroupConfig{
			groups["stormgrp"],
		},
		Services: []*ServiceConfig{
			services["beaconserver"],
			services["dam"],
			services["bagstorm"],
		},
	}

	groups["profilesearch"] = &ServiceGroupConfig{
		Name: "profilesearch",
		Groups: []*ServiceGroupConfig{
			groups["stormgrp"],
		},
		Services: []*ServiceConfig{
			services["ProfileSearchServer"],
		},
	}
}
