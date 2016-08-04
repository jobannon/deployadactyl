package interfaces

import "mime/multipart"

// Fetcher interface.
type Fetcher interface {
	Fetch(url, manifest string) (string, error)
	FetchLocal(file multipart.File) (string, error)
}
