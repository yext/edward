package generators

import "github.com/yext/edward/services"

type Generator interface {
	Name() string
	StartWalk(basePath string)
	StopWalk()
	VisitDir(path string) (bool, error)
	Err() error
	SetErr(err error)
}

type ServiceGenerator interface {
	Services() []*services.ServiceConfig
}

type GroupGenerator interface {
	Groups() []*services.ServiceGroupConfig
}

type ImportGenerator interface {
	Imports() []string
}

type generatorBase struct {
	err      error
	basePath string
}

func (e *generatorBase) Err() error {
	return e.err
}

func (e *generatorBase) SetErr(err error) {
	e.err = err
}

func (b *generatorBase) StartWalk(basePath string) {
	b.err = nil
	b.basePath = basePath
}
