package mocks

type Fetcher struct {
	FetchCall struct {
		Received struct {
			URL      string
			Manifest string
		}
		Returns struct {
			AppPath string
			Error   error
		}
	}
}

func (f *Fetcher) Fetch(url, manifest string) (string, error) {
	f.FetchCall.Received.URL = url
	f.FetchCall.Received.Manifest = manifest

	return f.FetchCall.Returns.AppPath, f.FetchCall.Returns.Error
}
