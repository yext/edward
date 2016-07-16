package reboot

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"

	"github.com/shirou/gopsutil/host"
	"github.com/yext/edward/common"
	"github.com/yext/errgo"
)

// hasRebooted checks the reboot marker under the given path and returns a boolean indicating whether the system has been rebooted since the last time the marker was set
func HasRebooted(markerPath string, logger common.Logger) (bool, error) {
	log := common.MaskLogger(logger)

	err := updateLegacyReboot(markerPath, logger)
	if err != nil {
		return false, errgo.Mask(err)
	}

	rebootFile := path.Join(markerPath, ".boottime")
	log.Printf("Checking reboot marker at: %v", rebootFile)
	if _, err := os.Stat(rebootFile); err != nil {
		if os.IsNotExist(err) {
			log.Printf("Reboot marker file not found.")
			return true, nil
		}
		return false, errgo.Mask(err)
	}

	rebootMarker, _ := ioutil.ReadFile(rebootFile)
	bootTime, err := host.BootTime()
	if err != nil {
		return false, errgo.Mask(err)
	}
	bootTimeStr := strconv.FormatUint(bootTime, 10)

	log.Printf("Reboot marker: '%v'. Syscall boot time: '%v'\n", string(rebootMarker), bootTimeStr)

	if bootTimeStr != string(rebootMarker) {
		log.Printf("Has rebooted since marker set.\n")
		return true, nil
	}

	log.Printf("No reboot since marker set.\n")
	return false, nil
}

func SetRebootMarker(markerPath string, logger common.Logger) error {
	log := common.MaskLogger(logger)

	rebootFile := path.Join(markerPath, ".boottime")
	bootTime, err := host.BootTime()
	if err != nil {
		return errgo.Mask(err)
	}
	bootTimeStr := strconv.FormatUint(bootTime, 10)
	log.Printf("Writing reboot time '%v' to: %v", bootTimeStr, rebootFile)
	err = ioutil.WriteFile(rebootFile, []byte(bootTimeStr), os.ModePerm)
	if err == nil {
		log.Printf("Reboot marker wrote successfully.\n")
	}
	return errgo.Mask(err)

}

func updateLegacyReboot(markerPath string, logger common.Logger) error {
	log := common.MaskLogger(logger)

	rebootFile := path.Join(markerPath, ".lastreboot")
	log.Printf("Checking for legacy reboot marker at: %v\n", rebootFile)

	if _, err := os.Stat(rebootFile); err != nil {
		if os.IsNotExist(err) {
			log.Printf("Legacy reboot file not found.")
		} else {
			log.Printf("Error reading legacy reboot file: %v", err)
		}

		return nil
	}

	rebootMarker, _ := ioutil.ReadFile(rebootFile)

	command := exec.Command("last", "-1", "reboot")
	output, err := command.CombinedOutput()
	if err != nil {
		return errgo.Mask(err)
	}
	log.Printf("Reboot marker: '%v'. Output from last: '%v'", rebootMarker, output)

	if string(output) == string(rebootMarker) {
		SetRebootMarker(markerPath, logger)
	}

	log.Printf("Deleting reboot file at: %v", rebootFile)
	_ = os.Remove(rebootFile)
	return nil
}
