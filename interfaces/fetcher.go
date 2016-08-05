package interfaces

import "mime/multipart"

// Fetcher interface.
type Fetcher interface {
	Fetch(url, manifest string) (string, error)
	FetchFromZip(file multipart.File) (string, error)
}
