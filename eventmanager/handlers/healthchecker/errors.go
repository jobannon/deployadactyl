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
	Domain  string
}

func (e MapRouteError) Error() string {
	return fmt.Sprintf("could not map temporary health check route %s.%s", e.AppName, e.Domain)
}

type ClientError struct {
	Err error
}

func (e ClientError) Error() string {
	return fmt.Sprintf("could not perform GET request: %s", e.Err.Error())
}

type LoginError struct {
	Output []byte
}

func (e LoginError) Error() string {
	return fmt.Sprintf("could not login")
}

type WrongEventTypeError struct {
	Type string
}

func (e WrongEventTypeError) Error() string {
	return fmt.Sprintf("wrong event type for healthchecker: %s", e.Type)
}
