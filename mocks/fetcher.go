package mocks

import "mime/multipart"

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
			File multipart.File
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

func (f *Fetcher) FetchFromZip(file multipart.File) (string, error) {
	f.FetchFromZipCall.Received.File = file

	return f.FetchCall.Returns.AppPath, f.FetchCall.Returns.Error
}
