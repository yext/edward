package updates

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/yext/edward/common"
)

func UpdateAvailable(repo, currentVersion, cachePath string, logger common.Logger) (bool, string, error) {
	output, err := exec.Command("git", "ls-remote", "-t", "git://"+repo).CombinedOutput()
	if err != nil {
		return false, "", errors.WithStack(err)
	}

	printf(logger, "Checking for cached version at %v", cachePath)
	isCached, latestVersion, err := getCachedVersion(cachePath)
	if err != nil {
		return false, "", errors.WithStack(err)
	}

	if !isCached {
		printf(logger, "No cached version, requesting from Git\n")
		latestVersion, err = findLatestVersionTag(output)
		if err != nil {
			return false, "", errors.WithStack(err)
		}
		printf(logger, "Caching version: %v", latestVersion)
		err = cacheVersion(cachePath, latestVersion)
		if err != nil {
			return false, "", errors.WithStack(err)
		}
	} else {
		printf(logger, "Found cached version\n")
	}

	printf(logger, "Comparing latest release %v, to current version %v\n", latestVersion, currentVersion)

	lv, err1 := version.NewVersion(latestVersion)
	cv, err2 := version.NewVersion(currentVersion)

	if err1 != nil {
		return false, latestVersion, errors.WithStack(err)
	}
	if err2 != nil {
		return true, latestVersion, errors.WithStack(err)
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

func getCachedVersion(cachePath string) (wasCached bool, cachedVersion string, err error) {
	info, err := os.Stat(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, "", nil
		}
		return false, "", errors.WithStack(err)
	}
	duration := time.Since(info.ModTime())
	if duration.Hours() >= 1 {
		return false, "", nil
	}
	content, err := ioutil.ReadFile(cachePath)
	if err != nil {
		return false, "", errors.WithStack(err)
	}
	return true, string(content), nil
}

func cacheVersion(cachePath, versionToCache string) error {
	err := ioutil.WriteFile(cachePath, []byte(versionToCache), 0644)
	return errors.WithStack(err)
}

func printf(logger common.Logger, f string, v ...interface{}) {
	if logger != nil {
		logger.Printf(f, v...)
	}
}
