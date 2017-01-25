package bluegreen

import "fmt"

type PushFailRollbackError struct {
	Err error
}

func (e PushFailRollbackError) Error() string {
	return fmt.Sprintf("push failed: rollback triggered: %s", e.Err)
}

type PushFailNoRollbackError struct{}

func (e PushFailNoRollbackError) Error() string {
	return "push failed: this is the first deploy, so no rollback occurred"
}
