package bluegreen

type PushFailRollbackError struct{}

func (e PushFailRollbackError) Error() string {
	return "push failed: rollback triggered"
}

type PushFailNoRollbackError struct{}

func (e PushFailNoRollbackError) Error() string {
	return "push failed: this is the first deploy, so no rollback occurred"
}
