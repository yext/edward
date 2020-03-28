package edward

import (
	"log"
	"os/exec"
	"path"
)

func (c *Client) telemetryEvent(params ...string) {
	if c.telemetryScript == "" {
		return
	}

	// Execute the script in the background
	go func() {
		cmd := exec.Command(path.Join(c.BasePath(), c.telemetryScript), params...)
		cmd.Dir = c.WorkingDir
		stdoutStderr, err := cmd.CombinedOutput()
		if err != nil {
			log.Println("Could not run telemetry script:", err)
		}
		log.Printf("Telemetry output:\n%s", stdoutStderr)
	}()
}
