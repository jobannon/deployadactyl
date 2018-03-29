package mocks

import (
	"io"
)

// Fetcher handmade mock for tests.
type Fetcher struct {
	FetchCall struct {
		Received struct {
			ArtifactURL string
			Manifest    string
		}
		Returns struct {
			AppPath string
			Error   error
		}
	}

	FetchFromZipCall struct {
		Received struct {
			Request io.Reader
		}
		Returns struct {
			AppPath string
			Error   error
		}
	}
}

// Fetch mock method.
func (f *Fetcher) Fetch(url, manifest string) (string, error) {
	f.FetchCall.Received.ArtifactURL = url
	f.FetchCall.Received.Manifest = manifest

	return f.FetchCall.Returns.AppPath, f.FetchCall.Returns.Error
}

// FetchZipFromRequest mock method.
func (f *Fetcher) FetchZipFromRequest(body io.Reader) (string, error) {
	f.FetchFromZipCall.Received.Request = body

	return f.FetchFromZipCall.Returns.AppPath, f.FetchFromZipCall.Returns.Error
}
