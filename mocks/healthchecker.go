package mocks

import "fmt"

// HealthChecker handmade mock for tests.
type HealthChecker struct {
	CheckCall struct {
		Received struct {
			Endpoint string
			URL      string
		}
		Returns struct {
			Error error
		}
	}
}

func (h *HealthChecker) Check(endpoint, serverURL string) error {
	h.CheckCall.Received.Endpoint = endpoint
	h.CheckCall.Received.URL = fmt.Sprintf("%s/%s", serverURL, endpoint)

	return h.CheckCall.Returns.Error
}
