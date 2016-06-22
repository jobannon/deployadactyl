package interfaces

type Fetcher interface {
	Fetch(url, manifest string) (string, error)
}
