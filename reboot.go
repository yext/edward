package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"

	"github.com/shirou/gopsutil/host"
	"github.com/yext/errgo"
)

func updateLegacyReboot(markerPath string) error {
	rebootFile := path.Join(markerPath, ".lastreboot")

	rebootMarker, _ := ioutil.ReadFile(rebootFile)

	command := exec.Command("last", "-1", "reboot")
	output, err := command.CombinedOutput()
	if err != nil {
		return errgo.Mask(err)
	}

	if string(output) == string(rebootMarker) {
		setRebootMarker(markerPath)
	}

	_ = os.Remove(rebootFile)
	return nil
}

// hasRebooted checks the reboot marker under the given path and returns a boolean indicating whether the system has been rebooted since the last time the marker was set
func hasRebooted(markerPath string) (bool, error) {
	err := updateLegacyReboot(markerPath)
	if err != nil {
		return false, errgo.Mask(err)
	}

	rebootFile := path.Join(markerPath, ".boottime")
	rebootMarker, _ := ioutil.ReadFile(rebootFile)
	bootTime, err := host.BootTime()
	if err != nil {
		return false, errgo.Mask(err)
	}
	bootTimeStr := strconv.FormatUint(bootTime, 10)

	if bootTimeStr != string(rebootMarker) {
		return true, nil
	}

	return false, nil
}

func setRebootMarker(markerPath string) error {
	rebootFile := path.Join(markerPath, ".boottime")
	bootTime, err := host.BootTime()
	if err != nil {
		return errgo.Mask(err)
	}
	bootTimeStr := strconv.FormatUint(bootTime, 10)
	err = ioutil.WriteFile(rebootFile, []byte(bootTimeStr), os.ModePerm)
	return errgo.Mask(err)

}
