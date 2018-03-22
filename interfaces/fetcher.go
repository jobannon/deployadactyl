package interfaces

import (
	"io"
)

// Fetcher interface.
type Fetcher interface {
	Fetch(url, manifest string) (string, error)
	FetchZipFromRequest(body io.Reader) (string, error)
}
