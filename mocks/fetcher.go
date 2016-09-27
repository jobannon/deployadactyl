package mocks

import "net/http"

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
			Request *http.Request
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

// FetchFromZip mock method.
func (f *Fetcher) FetchZipFromRequest(req *http.Request) (string, error) {
	f.FetchFromZipCall.Received.Request = req

	return f.FetchFromZipCall.Returns.AppPath, f.FetchFromZipCall.Returns.Error
}
