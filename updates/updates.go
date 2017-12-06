package updates

import (
	"context"
	"strings"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/yext/edward/common"
)

// UpdateAvailable determines if a newer version is available given a repo
func UpdateAvailable(owner, repo, currentVersion, cachePath string, logger common.Logger) (bool, string, error) {
	diskCache := diskcache.New(cachePath)
	transport := httpcache.NewTransport(diskCache)
	client := github.NewClient(transport.Client())

	release, _, err := client.Repositories.GetLatestRelease(context.Background(), owner, repo)
	if err != nil {
		// Log, but don't return rate limit errors
		if _, ok := err.(*github.RateLimitError); ok {
			printf(logger, "Rate limit error when requesting latest version %v", err)
			return false, "", nil
		}
		return false, "", errors.WithStack(err)
	}

	latestVersion := *release.TagName
	latestVersion = strings.Replace(latestVersion, "v", "", 1)

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

func printf(logger common.Logger, f string, v ...interface{}) {
	if logger != nil {
		logger.Printf(f, v...)
	}
}
