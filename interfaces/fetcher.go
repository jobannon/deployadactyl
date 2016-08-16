package interfaces

// Fetcher interface.
type Fetcher interface {
	Fetch(url, manifest string) (string, error)
	FetchFromZip(requestBody []byte) (string, error)
}
