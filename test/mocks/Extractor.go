package mocks

type Extractor struct {
	UnzipCall struct {
		Received struct {
			Source      string
			Destination string
			Manifest    string
		}
		Returns struct {
			Error error
		}
	}
}

func (e *Extractor) Unzip(source, destination, manifest string) error {
	e.UnzipCall.Received.Source = source
	e.UnzipCall.Received.Destination = destination
	e.UnzipCall.Received.Manifest = manifest

	return e.UnzipCall.Returns.Error
}
