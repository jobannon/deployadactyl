package interfaces

type Extractor interface {
	Unzip(string, string, string) error
}
