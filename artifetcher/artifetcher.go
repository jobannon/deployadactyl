// Package artifetcher downloads the artifact given a URL.
package artifetcher

import (
	"crypto/tls"
	"io"
	"net/http"
	"time"

	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-errors/errors"
	"github.com/op/go-logging"
	"github.com/spf13/afero"
)

const (
	cannotCreateTempFile      = "cannot create temp file"
	cannotGetURL              = "cannot GET url"
	cannotWriteResponseToFile = "cannot write response to file"
	cannotCreateTempDirectory = "cannot create temp directory"
	cannotUnzipArtifact       = "cannot unzip artifact"
)

// Artifetcher fetches artifacts within a file system with an Extractor.
type Artifetcher struct {
	FileSystem *afero.Afero
	Extractor  I.Extractor
	Log        *logging.Logger
}

// Fetch downloads an artifact located at URL.
// It then passes it to the extractor with the manifest for unzipping.
//
// Returns a string to the unzipped artifacts path and an error.
func (a *Artifetcher) Fetch(url, manifest string) (string, error) {
	a.Log.Debug("fetch URL: %s", url)

	artifactFile, err := a.FileSystem.TempFile("", "deployadactyl-")
	if err != nil {
		return "", errors.Errorf("%s: %s", cannotCreateTempFile, err)
	}
	defer artifactFile.Close()
	defer a.FileSystem.Remove(artifactFile.Name())

	var proxyClient = &http.Client{
		Transport: &http.Transport{
			Proxy:             http.ProxyFromEnvironment,
			DisableKeepAlives: true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			ResponseHeaderTimeout: 15 * time.Second,
		},
	}

	response, err := proxyClient.Get(url)
	if err != nil {
		return "", errors.Errorf("%s: %s: %s", cannotGetURL, url, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", errors.Errorf("%s: %s: %s", cannotGetURL, url, response.Status)
	}

	_, err = io.Copy(artifactFile, response.Body)
	if err != nil {
		a.Log.Debug("Response: %s", spew.Sdump(response))
		return "", errors.Errorf("%s: %s", cannotWriteResponseToFile, err)
	}

	unzippedPath, err := a.FileSystem.TempDir("", "deployadactyl-")
	if err != nil {
		return "", errors.Errorf("%s: %s", cannotCreateTempDirectory, err)
	}

	err = a.Extractor.Unzip(artifactFile.Name(), unzippedPath, manifest)
	if err != nil {
		a.FileSystem.RemoveAll(unzippedPath)
		return "", errors.Errorf("%s: %s", cannotUnzipArtifact, err)
	}

	a.Log.Info("successfully unzipped to tempdir %s", unzippedPath)
	return unzippedPath, nil
}
