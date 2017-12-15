package commandline

import (
	"os"
	"os/exec"
	"path"

	"github.com/pkg/errors"
)

// ConstructCommand creates an exec.Cmd from path components, command as a single string and an environment var function
func ConstructCommand(workingDir string, targetPath *string, command string, getenv func(string) string) (*exec.Cmd, error) {
	command, cmdArgs, err := ParseCommand(os.Expand(command, getenv))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	cmd := exec.Command(command, cmdArgs...)
	cmd.Dir = BuildAbsPath(workingDir, targetPath)
	return cmd, nil
}

// BuildAbsPath will ensure the targetPath is absolute, joining to workingDir
// if necessary.
func BuildAbsPath(workingDir string, targetPath *string) string {
	if targetPath != nil {
		expandedPath := os.ExpandEnv(*targetPath)
		if !path.IsAbs(expandedPath) {
			return path.Join(workingDir, expandedPath)
		}
		*targetPath = expandedPath
		return *targetPath
	}
	return workingDir
}
