package interfaces

import "net/http"

// Fetcher interface.
type Fetcher interface {
	Fetch(url, manifest string) (string, error)
	FetchFromZip(*http.Request) (string, error)
}
