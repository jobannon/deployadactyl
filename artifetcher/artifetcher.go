// Package artifetcher downloads the artifact given a URL.
package artifetcher

import (
	"io"
	"net"
	"net/http"
	"time"

	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/spf13/afero"
)

type ArtifetcherConstructor func(fs *afero.Afero, ex I.Extractor, log I.DeploymentLogger) I.Fetcher

func NewArtifetcher(fs *afero.Afero, ex I.Extractor, log I.DeploymentLogger) I.Fetcher {
	return &Artifetcher{
		FileSystem: fs,
		Extractor:  ex,
		Log:        log,
	}
}

// Artifetcher fetches artifacts within a file system with an Extractor.
type Artifetcher struct {
	FileSystem *afero.Afero
	Extractor  I.Extractor
	Log        I.DeploymentLogger
}

// Fetch downloads an artifact located at URL.
// It then passes it to the extractor with the manifest for unzipping.
//
// Returns a string to the unzipped artifacts path and an error.
func (a *Artifetcher) Fetch(url, manifest string) (string, error) {
	a.Log.Info("fetching artifact")
	a.Log.Debugf("artifact URL: %s", url)

	artifactFile, err := a.FileSystem.TempFile("", "deployadactyl-zip-")
	if err != nil {
		return "", CreateTempFileError{err}
	}
	defer artifactFile.Close()
	defer a.FileSystem.Remove(artifactFile.Name())

	var client = &http.Client{
		Timeout: 15 * time.Minute,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   60 * time.Second,
				KeepAlive: 60 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   15 * time.Second,
			ResponseHeaderTimeout: 15 * time.Second,
			ExpectContinueTimeout: 2 * time.Second,
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", FetcherRequestError{err}
	}

	response, err := client.Do(req)
	if err != nil {
		return "", GetUrlError{url, err}
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", GetStatusError{url, response.Status}
	}

	_, err = io.Copy(artifactFile, response.Body)
	if err != nil {
		return "", WriteResponseError{err}
	}

	unzippedPath, err := a.FileSystem.TempDir("", "deployadactyl-unzipped-")
	if err != nil {
		return "", CreateTempDirectoryError{err}
	}

	err = a.Extractor.Unzip(artifactFile.Name(), unzippedPath, manifest)
	if err != nil {
		a.FileSystem.RemoveAll(unzippedPath)
		return "", UnzipError{err}

	}

	a.Log.Debugf("fetched and unzipped to tempdir: %s", unzippedPath)
	return unzippedPath, nil
}

// FetchZipFromRequest fetches files from a compressed zip file in the request body.
//
// Returns a string to the unzipped application path and an error.
func (a *Artifetcher) FetchZipFromRequest(body io.Reader) (string, string, error) {

	zipFile, err := a.FileSystem.TempFile("", "deployadactyl-")
	if err != nil {
		return "", "", CreateTempFileError{err}
	}
	defer zipFile.Close()
	defer a.FileSystem.Remove(zipFile.Name())

	a.Log.Infof("fetching zip file %s", zipFile.Name())

	_, err = io.Copy(zipFile, body)
	if err != nil {
		return "", "", WriteResponseError{err}
	}

	unzippedPath, err := a.FileSystem.TempDir("", "deployadactyl-")
	if err != nil {
		return "", "", CreateTempDirectoryError{err}
	}

	err = a.Extractor.Unzip(zipFile.Name(), unzippedPath, "")
	if err != nil {
		a.FileSystem.RemoveAll(unzippedPath)
		return "", "", UnzipError{err}
	}

	manifest, err := a.FileSystem.ReadFile(unzippedPath + "/manifest.yml")
	if err != nil {
		return "", "", err
	}

	a.Log.Debugf("fetched and unzipped to tempdir %s", unzippedPath)
	return unzippedPath, string(manifest), nil
}
