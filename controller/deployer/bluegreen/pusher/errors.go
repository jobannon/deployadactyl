package pusher

import "fmt"

type CloudFoundryGetLogsError struct {
	CfTaskErr error
	CfLogErr  error
}

func (e CloudFoundryGetLogsError) Error() string {
	return fmt.Sprintf("%s: cannot get Cloud Foundry logs: %s", e.CfTaskErr, e.CfLogErr)
}

type DeleteApplicationError struct {
	ApplicationName string
	Out             []byte
}

func (e DeleteApplicationError) Error() string {
	return fmt.Sprintf("cannot delete %s: %s", e.ApplicationName, string(e.Out))
}

type LoginError struct {
	FoundationURL string
	Out           []byte
}

func (e LoginError) Error() string {
	return fmt.Sprintf("cannot login to %s: %s", e.FoundationURL, string(e.Out))
}

type RenameError struct {
	ApplicationName string
	Out             []byte
}

func (e RenameError) Error() string {
	return fmt.Sprintf("cannot rename %s: %s", e.ApplicationName, string(e.Out))
}

type PushError struct{}

func (e PushError) Error() string {
	return "check the Cloud Foundry output above for more information"
}

type MapRouteError struct{}

func (e MapRouteError) Error() string {
	return "map route failed: check the Cloud Foundry output above for more information"
}

type UnmapRouteError struct {
	ApplicationName string
}

func (e UnmapRouteError) Error() string {
	return fmt.Sprintf("failed to unmap route for %s", e.ApplicationName)
}
