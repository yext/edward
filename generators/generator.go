package generators

import "github.com/yext/edward/services"

// Generator provides an interface to identify a generator and perform a directory walk
// to find projects for configuration.
type Generator interface {
	Name() string
	StartWalk(basePath string)
	StopWalk()
	VisitDir(path string) (bool, error)
	Err() error
	SetErr(err error)
}

// ServiceGenerator provides an interface to return a slice of services after performing
// a directory walk.
type ServiceGenerator interface {
	Services() []*services.ServiceConfig
}

// GroupGenerator provides an interface to return a slice of groups after performing
// a directory walk.
type GroupGenerator interface {
	Groups() []*services.ServiceGroupConfig
}

// ImportGenerator provides an interface to return a slice of Edward config file paths
// to be imported after a directory walk.
type ImportGenerator interface {
	Imports() []string
}

type generatorBase struct {
	err      error
	basePath string
}

// Err returns the most recent error from this generator
func (b *generatorBase) Err() error {
	return b.err
}

// SetErr allows an error to be applied to this generator from outside
func (b *generatorBase) SetErr(err error) {
	b.err = err
}

// StartWalk lets a generator know that a directory walk has been started, with the
// given starting path
func (b *generatorBase) StartWalk(basePath string) {
	b.err = nil
	b.basePath = basePath
}

// StopWalk lets a generator know that a directory walk has been completed, so it
// can perform any necessary cleanup or consolidation.
func (b *generatorBase) StopWalk() {
}
