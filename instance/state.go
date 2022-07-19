package instance

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/yext/edward/services"
)

type State string

const (
	StateStarting State = "STARTING"
	StateRunning  State = "RUNNING"
	StateStopped  State = "STOPPED"
	StateDied     State = "DIED"
	StateUnknown  State = "UNKNOWN"
)

type Status struct {
	State       State                   `json:"status"`
	Ports       []string                `json:"ports"` // Ports opened by this instance
	StdoutLines int                     `json:"stdoutLines"`
	StderrLines int                     `json:"stderrLines"`
	StartTime   time.Time               `json:"startTime"`
	MemoryInfo  *process.MemoryInfoStat `json:"memoryInfo,omitempty"`
}

var store = make(map[string]Status)

func LoadStatusForService(service *services.ServiceConfig, baseDir string) (map[string]Status, error) {
	var statuses = make(map[string]Status)
	statusDir := statusDir(service, baseDir)
	if _, err := os.Stat(statusDir); os.IsNotExist(err) {
		return statuses, nil
	}
	files, err := ioutil.ReadDir(statusDir)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for _, f := range files {
		raw, err := ioutil.ReadFile(path.Join(statusDir, f.Name()))
		if err != nil {
			continue
		}

		if len(raw) == 0 {
			statuses[f.Name()] = Status{
				State: StateUnknown,
			}
			continue
		}

		var s Status
		err = json.Unmarshal(raw, &s)
		if err == nil {
			statuses[f.Name()] = s
		}
	}
	return statuses, nil
}

func SaveStatusForService(service *services.ServiceConfig, instanceId string, status Status, baseDir string) error {
	err := createStatusDirIfNeeded(service, baseDir)
	if err != nil {
		return errors.WithStack(err)
	}

	// Save status file
	bytes, err := json.Marshal(status)
	if err != nil {
		return errors.WithMessage(err, "marshal status")
	}
	statusFile := statusFile(service, baseDir, instanceId)
	err = ioutil.WriteFile(statusFile, bytes, os.ModePerm)
	if err != nil {
		return errors.WithMessage(err, "save status")
	}
	return nil
}

func DeleteAllStatusesForService(service *services.ServiceConfig, baseDir string) error {
	statusDir := statusDir(service, baseDir)
	if _, err := os.Stat(statusDir); os.IsNotExist(err) {
		return nil
	}
	err := os.RemoveAll(statusDir)
	return errors.WithStack(err)
}

func DeleteStatusForService(service *services.ServiceConfig, instanceId, baseDir string) error {
	err := createStatusDirIfNeeded(service, baseDir)
	if err != nil {
		return errors.WithStack(err)
	}
	statusFile := statusFile(service, baseDir, instanceId)
	err = os.Remove(statusFile)
	return errors.WithStack(err)
}

func createStatusDirIfNeeded(service *services.ServiceConfig, baseDir string) error {
	// Create status dir as required
	statusDir := statusDir(service, baseDir)
	if _, err := os.Stat(statusDir); os.IsNotExist(err) {
		err := os.Mkdir(statusDir, os.ModePerm)
		if err != nil {
			return errors.WithMessage(err, "create status directory")
		}
	}
	return nil
}

func statusDir(service *services.ServiceConfig, baseDir string) string {
	return path.Join(baseDir, service.IdentifyingFilename())
}

func statusFile(service *services.ServiceConfig, baseDir string, instanceId string) string {
	return path.Join(statusDir(service, baseDir), fmt.Sprintf("%v", instanceId))
}
