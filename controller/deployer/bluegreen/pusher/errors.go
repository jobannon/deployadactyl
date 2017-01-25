package pusher

import "fmt"

type RenameFailError struct {
	Err error
}

func (e RenameFailError) Error() string {
	return fmt.Sprintf("rename failed: %s", e.Err)
}

type CloudFoundryGetLogsError struct {
	CfTaskErr error
	CfLogErr  error
}

func (e CloudFoundryGetLogsError) Error() string {
	return fmt.Sprintf("%s: cannot get Cloud Foundry logs: %s", e.CfTaskErr, e.CfLogErr)
}

type DeleteApplicationError struct {
	ApplicationName string
	Err             error
}

func (e DeleteApplicationError) Error() string {
	return fmt.Sprintf("cannot delete %s: %s", e.ApplicationName, e.Err)
}

type LoginError struct {
	FoundationURL string
	Err           error
}

func (e LoginError) Error() string {
	return fmt.Sprintf("cannot login to %s: %s", e.FoundationURL, e.Err)
}

type RenameApplicationError struct {
	ApplicationName string
	Err             error
}

func (e RenameApplicationError) Error() string {
	return fmt.Sprintf("cannot rename %s: %s", e.ApplicationName, e.Err)
}
