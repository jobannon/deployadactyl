package artifetcher

import "fmt"

type CreateTempFileError struct {
	Err error
}

func (e CreateTempFileError) Error() string {
	return fmt.Sprintf("cannot create temp file: %s", e.Err)
}

type FetcherRequestError struct {
	Err error
}

func (e FetcherRequestError) Error() string {
	return fmt.Sprintf("cannot create artifact fetch request: %s", e.Err)
}

type GetUrlError struct {
	Url string
	Err error
}

func (e GetUrlError) Error() string {
	return fmt.Sprintf("cannot GET url: %s: %s", e.Url, e.Err)
}

type GetStatusError struct {
	Url    string
	Status string
}

func (e GetStatusError) Error() string {
	return fmt.Sprintf("cannot GET url: %s: %s", e.Url, e.Status)
}

type WriteResponseError struct {
	Err error
}

func (e WriteResponseError) Error() string {
	return fmt.Sprintf("cannot write response to file: %s", e.Err)
}

type CreateTempDirectoryError struct {
	Err error
}

func (e CreateTempDirectoryError) Error() string {
	return fmt.Sprintf("cannot create temp directory: %s", e.Err)
}

type UnzipError struct {
	Err error
}

func (e UnzipError) Error() string {
	return fmt.Sprintf("cannot unzip artifact: %s", e.Err)
}
