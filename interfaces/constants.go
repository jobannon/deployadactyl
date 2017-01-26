package interfaces

const (
	ENV_VARS_FOUND_EVENT                                         = "environment.variables.found"
	DEPLOY_START_EVENT                                           = "deploy.start"
	DEPLOY_FINISH_EVENT                                          = "deploy.finish"
	DEPLOY_SUCCESS_EVENT                                         = "deploy.success"
	DEPLOY_FAILURE_EVENT                                         = "deploy.failure"
	ENABLE_ENV_VAR_HANDLER_FLAG_ARG                              = "env"
	ENABLE_DISABLE_FILESYSTEM_CLEANUP_ON_DEPLOY_FAILURE_FLAG_ARG = "cleanOnFail"
)
