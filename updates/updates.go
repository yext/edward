package updates

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"regexp"

	"github.com/hashicorp/go-version"
	"github.com/yext/edward/common"
	"github.com/yext/errgo"
)

func UpdateAvailable(repo, currentVersion, cachePath string, logger common.Logger) (bool, string, error) {
	output, err := exec.Command("git", "ls-remote", "-t", "git://"+repo).CombinedOutput()
	if err != nil {
		return false, "", errgo.Mask(err)
	}

	// TODO: Cache this result
	latestVersion, err := findLatestVersionTag(output)
	if err != nil {
		return false, "", errgo.Mask(err)
	}

	lv, err1 := version.NewVersion(latestVersion)
	cv, err2 := version.NewVersion(currentVersion)

	if err1 != nil {
		return false, latestVersion, errgo.Mask(err)
	}
	if err2 != nil {
		return true, latestVersion, errgo.Mask(err)
	}

	return cv.LessThan(lv), latestVersion, nil
}

func findLatestVersionTag(refs []byte) (string, error) {
	r := bytes.NewReader(refs)
	reader := bufio.NewReader(r)
	line, isPrefix, err := reader.ReadLine()

	var greatestVersion string

	var validID = regexp.MustCompile(`([0-9]+\.[0-9]+\.[0-9])?$`)
	for err != io.EOF {
		if isPrefix {
			fmt.Println("Prefix")
		}
		match := validID.FindString(string(line))
		if len(match) > 0 && match > greatestVersion {
			greatestVersion = match
		}
		line, isPrefix, err = reader.ReadLine()
	}
	return greatestVersion, nil
}
