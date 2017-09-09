package updates

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/yext/edward/common"
)

// UpdateAvailable determines if a newer version is available given a repo
func UpdateAvailable(owner, repo, currentVersion, cachePath string, logger common.Logger) (bool, string, error) {
	printf(logger, "Checking for cached version at %v", cachePath)
	isCached, latestVersion, err := getCachedVersion(cachePath)
	if err != nil {
		return false, "", errors.WithStack(err)
	}

	if !isCached {
		printf(logger, "No cached version, requesting from Git\n")
		client := github.NewClient(nil)

		release, _, err := client.Repositories.GetLatestRelease(context.Background(), owner, repo)
		if err != nil {
			// Log, but don't return rate limit errors
			if _, ok := err.(*github.RateLimitError); ok {
				printf(logger, "Rate limit error when requesting latest version %v", err)
				return false, "", nil
			}
			return false, "", errors.WithStack(err)
		}

		latestVersion = *release.TagName
		latestVersion = strings.Replace(latestVersion, "v", "", 1)
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
