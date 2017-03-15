package healthchecker

import "fmt"

type HealthCheckError struct {
	Endpoint string
}

func (e HealthCheckError) Error() string {
	return fmt.Sprintf("health check failed for endpoint: %s", e.Endpoint)
}

type MapRouteError struct {
	AppName string
}

func (e MapRouteError) Error() string {
	return fmt.Sprintf("could not map temporary health check route to %s", e.AppName)
}

type ClientError struct {
	Err error
}

func (e ClientError) Error() string {
	return fmt.Sprintf("could not perform GET request: %s", e.Err.Error())
}
