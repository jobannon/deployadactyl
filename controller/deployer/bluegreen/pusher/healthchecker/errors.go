package healthchecker

import "fmt"

type HealthCheckError struct {
	Endpoint string
}

func (e HealthCheckError) Error() string {
	return fmt.Sprintf("health check failed for endpoint: %s", e.Endpoint)
}
