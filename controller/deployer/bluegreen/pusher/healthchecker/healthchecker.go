package healthchecker

import (
	"fmt"
	"net/http"
	"strings"
)

// HealthChecker will check an endpoint for a http.StatusOK
type HealthChecker struct{}

// Check takes an endpoint and a serverURL. It does an http.Get to get the response
// status and return an error if it is not http.StatusOK.
func (h HealthChecker) Check(endpoint, serverURL string) error {

	trimmedEndpoint := strings.TrimPrefix(endpoint, "/")
	resp, err := http.Get(fmt.Sprintf("%s/%s", serverURL, trimmedEndpoint))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return HealthCheckError{endpoint}
	}

	return nil
}
