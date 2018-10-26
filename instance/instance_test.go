package instance

import (
	"syscall"
	"testing"

	"github.com/theothertomelliott/must"
	"github.com/yext/edward/common"
	"github.com/yext/edward/home"
	"github.com/yext/edward/services"
	"github.com/yext/edward/services/backends/fake"
)

func TestLoad(t *testing.T) {
	var tests = []struct {
		name        string
		dirConfig   *home.EdwardConfiguration
		processes   Processes
		service     *services.ServiceConfig
		overrides   services.ContextOverride
		expected    *Instance
		expectedErr error
	}{
		{
			name:      "handles lack of state file",
			processes: &fakeProcesses{exists: true},
			dirConfig: &home.EdwardConfiguration{
				PidDir:   "testdata/emptydir",
				StateDir: "testdata/emptydir",
			},
			service: &services.ServiceConfig{
				Name:       "testService",
				ConfigFile: "/path/to/service",
			},
			expected: &Instance{
				ConfigFile:    "/path/to/service",
				EdwardVersion: common.EdwardVersion,
			},
		},
		{
			name:      "loads basic state file",
			processes: &fakeProcesses{exists: true},
			dirConfig: &home.EdwardConfiguration{
				PidDir:   "testdata/basicstatus",
				StateDir: "testdata/basicstatus",
			},
			service: &services.ServiceConfig{
				Name:       "testService",
				ConfigFile: "/path/to/service",
			},
			expected: &Instance{
				Pid:           102,
				ConfigFile:    "/path/to/service",
				EdwardVersion: common.EdwardVersion,
			},
		},
		{
			name:      "marks PID as zero when not matched",
			processes: &fakeProcesses{exists: false},
			dirConfig: &home.EdwardConfiguration{
				PidDir:   "testdata/basicstatus",
				StateDir: "testdata/basicstatus",
			},
			service: &services.ServiceConfig{
				Name:       "testService",
				ConfigFile: "/path/to/service",
			},
			expected: &Instance{
				Pid:           0,
				ConfigFile:    "/path/to/service",
				EdwardVersion: common.EdwardVersion,
			},
		},
		{
			name:      "loads legacy PIDs",
			processes: &fakeProcesses{exists: true},
			dirConfig: &home.EdwardConfiguration{
				PidDir: "testdata/legacypidfile",
			},
			service: &services.ServiceConfig{
				Name: "testService",
			},
			expected: &Instance{
				EdwardVersion: common.EdwardVersion,
				Pid:           101,
			},
		},
	}

	services.RegisterBackend(&fake.Loader{})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Load(test.dirConfig, test.processes, test.service, test.overrides)
			test.expected.Service = got.Service
			test.expected.dirConfig = test.dirConfig
			test.expected.processes = test.processes
			got.InstanceId = ""
			must.BeEqual(t, test.expected, got)
			must.BeEqualErrors(t, test.expectedErr, err)
		})
	}
}

type fakeProcesses struct {
	exists bool
}

func (p *fakeProcesses) SendSignal(pid int, signal syscall.Signal) error {
	return nil
}

func (p *fakeProcesses) KillGroup(pid int, sudo bool) error {
	return nil
}

func (p *fakeProcesses) PidExists(pid int) (bool, error) {
	return p.exists, nil
}

func (p *fakeProcesses) PidCommandMatches(pid int, value string) (bool, error) {
	return p.exists, nil
}
