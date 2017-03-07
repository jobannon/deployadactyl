package healthchecker_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	. "github.com/compozed/deployadactyl/controller/deployer/bluegreen/pusher/healthchecker"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Healthchecker", func() {

	var requestURL string

	Describe("checking the health of an endpoint", func() {
		Context("when the endpoint returns a http.StatusOK", func() {
			It("does not return an error", func() {
				endpoint := "/health"

				testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requestURL = r.RequestURI
				}))
				defer testServer.Close()

				err := Check(endpoint, testServer.URL)
				Expect(err).ToNot(HaveOccurred())

				Expect(requestURL).To(Equal(endpoint))
			})
		})

		Context("when the endpoint does not return a http.StatusOK ", func() {
			It("returns an error", func() {
				endpoint := "/health"

				testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
					requestURL = r.RequestURI
				}))
				defer testServer.Close()

				err := Check(endpoint, testServer.URL)
				Expect(err).To(MatchError(HealthCheckError{}))

				Expect(requestURL).To(Equal(endpoint))
			})
		})
	})

	Describe("format of endpoint parameter", func() {
		Context("when the endpoint does not include a '/'", func() {
			It("adds the leading '/'", func() {
				endpoint := "health"

				testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requestURL = fmt.Sprintf("http://%s%s", r.Host, r.RequestURI)
				}))
				defer testServer.Close()

				err := Check(endpoint, testServer.URL)
				Expect(err).ToNot(HaveOccurred())

				Expect(requestURL).To(Equal(fmt.Sprintf("%s/%s", testServer.URL, endpoint)))
			})
		})

		Context("when the endpoint does include a '/'", func() {
			It("adds the leading '/'", func() {
				endpoint := "/health"

				testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requestURL = fmt.Sprintf("http://%s%s", r.Host, r.RequestURI)
				}))
				defer testServer.Close()

				err := Check(endpoint, testServer.URL)
				Expect(err).ToNot(HaveOccurred())

				Expect(requestURL).To(Equal(fmt.Sprintf("%s%s", testServer.URL, endpoint)))
			})
		})
	})
})
