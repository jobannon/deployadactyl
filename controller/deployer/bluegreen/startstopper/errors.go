package startstopper

import "fmt"

type StartError struct {
	ApplicationName string
	Out             []byte
}

func (e StartError) Error() string {
	return fmt.Sprintf("cannot start %s: %s", e.ApplicationName, string(e.Out))
}

type StopError struct {
	ApplicationName string
	Out             []byte
}

func (e StopError) Error() string {
	return fmt.Sprintf("cannot stop %s: %s", e.ApplicationName, string(e.Out))
}

type ExistsError struct {
	ApplicationName string
}

func (e ExistsError) Error() string {
	return fmt.Sprintf("app %s doesn't exist", e.ApplicationName)
}
