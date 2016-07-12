package generators

import (
	"fmt"

	"github.com/yext/edward/services"
)

type ConfigGenerator func(path string) ([]*services.ServiceConfig, []*services.ServiceGroupConfig, error)

var Generators map[string]ConfigGenerator

func RegisterGenerator(name string, generator ConfigGenerator) {
	if Generators == nil {
		Generators = make(map[string]ConfigGenerator)
	}
	Generators[name] = generator
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
