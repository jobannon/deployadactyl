package interfaces

// Extractor interface.
type Extractor interface {
	Unzip(source, destination, manifest string) error
}
