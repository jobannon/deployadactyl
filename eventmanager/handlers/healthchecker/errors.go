package healthchecker

import (
	"fmt"
)

type HealthCheckError struct {
	Endpoint string
	Body     []byte
}

func (e HealthCheckError) Error() string {
	return fmt.Sprintf("health check returned %s for endpoint %s", e.Body, e.Endpoint)
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
	FoundationURL string
}

func (e LoginError) Error() string {
	return fmt.Sprintf("could not login to %s", e.FoundationURL)
}

type WrongEventTypeError struct {
	Type string
}

func (e WrongEventTypeError) Error() string {
	return fmt.Sprintf("wrong event type for healthchecker: %s", e.Type)
}
