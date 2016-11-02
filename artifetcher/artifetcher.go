// Package artifetcher downloads the artifact given a URL.
package artifetcher

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/op/go-logging"
	"github.com/spf13/afero"
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
	a.Log.Info("fetching artifact")
	a.Log.Debug("artifact URL: %s", url)

	artifactFile, err := a.FileSystem.TempFile("", "deployadactyl-zip-")
	if err != nil {
		return "", fmt.Errorf("cannot create temp file: %s", err)
	}
	defer artifactFile.Close()
	defer a.FileSystem.Remove(artifactFile.Name())

	var client = &http.Client{
		Timeout: 4 * time.Minute,
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
		return "", fmt.Errorf("cannot create artifactory request: %s", err)
	}

	response, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("cannot GET url: %s: %s", url, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cannot GET url: %s: %s", url, response.Status)
	}

	_, err = io.Copy(artifactFile, response.Body)
	if err != nil {
		return "", fmt.Errorf("cannot write response to file: %s", err)
	}

	unzippedPath, err := a.FileSystem.TempDir("", "deployadactyl-unzipped-")
	if err != nil {
		return "", fmt.Errorf("cannot create temp directory: %s", err)
	}

	err = a.Extractor.Unzip(artifactFile.Name(), unzippedPath, manifest)
	if err != nil {
		a.FileSystem.RemoveAll(unzippedPath)
		return "", fmt.Errorf("cannot unzip artifact: %s", err)
	}

	a.Log.Debug("fetched and unzipped to tempdir: %s", unzippedPath)
	return unzippedPath, nil
}

// FetchZipFromRequest fetches files from a compressed zip file in the request body.
//
// Returns a string to the unzipped application path and an error.
func (a *Artifetcher) FetchZipFromRequest(req *http.Request) (string, error) {
	zipFile, err := a.FileSystem.TempFile("", "deployadactyl-")
	if err != nil {
		return "", fmt.Errorf("cannot create temp file: %s", err)
	}
	defer zipFile.Close()
	defer a.FileSystem.Remove(zipFile.Name())

	a.Log.Info("fetching zip file %s", zipFile.Name())

	if _, err = io.Copy(zipFile, req.Body); err != nil {
		return "", fmt.Errorf("cannot write response to file: %s", err)
	}

	unzippedPath, err := a.FileSystem.TempDir("", "deployadactyl-")
	if err != nil {
		return "", fmt.Errorf("cannot create temp directory: %s", err)
	}

	err = a.Extractor.Unzip(zipFile.Name(), unzippedPath, "")
	if err != nil {
		a.FileSystem.RemoveAll(unzippedPath)
		return "", fmt.Errorf("cannot unzip artifact: %s", err)
	}

	a.Log.Debug("fetched and unzipped to tempdir %s", unzippedPath)
	return unzippedPath, nil
}
