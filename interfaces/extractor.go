package interfaces

type Extractor interface {
	Unzip(source, destination, manifest string) error
}
