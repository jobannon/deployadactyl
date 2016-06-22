package prechecker_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/compozed/deployadactyl/config"
	. "github.com/compozed/deployadactyl/deployer/prechecker"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/compozed/deployadactyl/test/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Prechecker", func() {
	Describe("AssertAllFoundationsUp", func() {
		var (
			httpStatus        int
			err               error
			foundationApiURLs []string
			prechecker        Prechecker
			eventManager      *mocks.EventManager
			configServer      *httptest.Server
			environment       config.Environment
			event             S.Event
		)

		BeforeEach(func() {
			foundationApiURLs = []string{}

			eventManager = &mocks.EventManager{}
			prechecker = Prechecker{eventManager}

		})

		JustBeforeEach(func() {
			configServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				foundationApiURLs = append(foundationApiURLs, r.URL.Path)
				w.WriteHeader(httpStatus)
			}))

			environment = config.Environment{
				Foundations: []string{configServer.URL},
			}
		})

		AfterEach(func() {
			configServer.Close()
		})

		Context("when no foundations are given", func() {
			JustBeforeEach(func() {
				environment.Foundations = nil
			})

			It("returns an error and emits an event", func() {
				precheckerEventData := S.PrecheckerEventData{
					Environment: environment,
					Description: "no foundations configured",
				}

				event = S.Event{
					Type: "validate.foundationsUnavailable",
					Data: precheckerEventData,
				}
				eventManager.On("Emit", event).Return(nil).Times(1)

				Expect(prechecker.AssertAllFoundationsUp(environment)).ToNot(Succeed())
			})
		})

		Context("when all foundations are up", func() {
			BeforeEach(func() {
				httpStatus = http.StatusOK
			})

			It("returns a nil error", func() {
				err = prechecker.AssertAllFoundationsUp(environment)

				Expect(err).ToNot(HaveOccurred())
				Expect(foundationApiURLs).To(ConsistOf("/v2/info"))
			})
		})

		Context("when the http returns a 500", func() {
			BeforeEach(func() {
				httpStatus = http.StatusInternalServerError
			})

			It("returns an error and emits an event", func() {
				precheckerEventData := S.PrecheckerEventData{
					Environment: environment,
					Description: "deploy aborted, one or more CF foundations unavailable",
				}

				event = S.Event{
					Type: "validate.foundationsUnavailable",
					Data: precheckerEventData,
				}
				eventManager.On("Emit", event).Return(nil).Times(1)

				err = prechecker.AssertAllFoundationsUp(environment)

				Expect(err).To(HaveOccurred())
				Expect(foundationApiURLs).To(ConsistOf("/v2/info"))
			})
		})

		Context("when a foundation is down", func() {
			BeforeEach(func() {
				httpStatus = http.StatusNotFound
			})

			It("returns an error and emits an event", func() {
				precheckerEventData := S.PrecheckerEventData{
					Environment: environment,
					Description: "deploy aborted, one or more CF foundations unavailable",
				}
				event = S.Event{
					Type: "validate.foundationsUnavailable",
					Data: precheckerEventData,
				}
				eventManager.On("Emit", event).Return(nil).Times(1)

				err = prechecker.AssertAllFoundationsUp(environment)

				Expect(err).To(HaveOccurred())
			})
		})
	})
})
