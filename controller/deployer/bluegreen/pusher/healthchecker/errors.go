package healthchecker

import "fmt"

type HealthCheckError struct{}

func (e HealthCheckError) Error() string {
	return fmt.Sprintf("health check failed")
}
