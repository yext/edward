package updates

import (
	"context"
	"log"
	"strings"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
)

// UpdateAvailable determines if a newer version is available given a repo
func UpdateAvailable(owner, repo, currentVersion, cachePath string) (bool, string, error) {
	diskCache := diskcache.New(cachePath)
	transport := httpcache.NewTransport(diskCache)
	client := github.NewClient(transport.Client())

	release, _, err := client.Repositories.GetLatestRelease(context.Background(), owner, repo)
	if err != nil {
		// Log, but don't return rate limit errors
		if _, ok := err.(*github.RateLimitError); ok {
			log.Printf("Rate limit error when requesting latest version %v", err)
			return false, "", nil
		}
		return false, "", errors.WithStack(err)
	}

	latestVersion := *release.TagName
	latestVersion = strings.Replace(latestVersion, "v", "", 1)

	log.Printf("Comparing latest release %v, to current version %v\n", latestVersion, currentVersion)

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
