package artifetcher

import (
	"crypto/tls"
	"io"
	"net/http"
	"time"

	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/go-errors/errors"
	"github.com/op/go-logging"
	"github.com/spf13/afero"
)

const (
	cannotCreateTempFile      = "cannot create temp file"
	cannotGetUrl              = "cannot GET url"
	cannotWriteResponseToFile = "cannot write response to file"
	cannotCreateTempDirectory = "cannot create temp directory"
	cannotUnzipArtifact       = "cannot unzip artifact"
)

type Artifetcher struct {
	FileSystem *afero.Afero
	Extractor  I.Extractor
	Log        *logging.Logger
}

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
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			ResponseHeaderTimeout: 15 * time.Second,
		},
	}

	response, err := proxyClient.Get(url)
	if err != nil {
		return "", errors.Errorf("%s: %s: %s", cannotGetUrl, url, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", errors.Errorf("%s: %s: %s", cannotGetUrl, url, response.Status)
	}

	_, err = io.Copy(artifactFile, response.Body)
	if err != nil {
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
