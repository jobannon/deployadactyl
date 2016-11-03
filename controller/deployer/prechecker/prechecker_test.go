package prechecker_test

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/compozed/deployadactyl/config"
	. "github.com/compozed/deployadactyl/controller/deployer/prechecker"
	"github.com/compozed/deployadactyl/mocks"
	S "github.com/compozed/deployadactyl/structs"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Prechecker", func() {
	Describe("AssertAllFoundationsUp", func() {
		var (
			httpStatus     int
			foundationURls []string
			prechecker     Prechecker
			eventManager   *mocks.EventManager
			testServer     *httptest.Server
			environment    config.Environment
			event          S.Event
		)

		BeforeEach(func() {
			foundationURls = []string{}

			eventManager = &mocks.EventManager{}
			prechecker = Prechecker{EventManager: eventManager}

			testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				foundationURls = append(foundationURls, r.URL.Path)
				w.WriteHeader(httpStatus)
			}))

			environment = config.Environment{
				Foundations: []string{testServer.URL},
			}
		})

		AfterEach(func() {
			testServer.Close()
		})

		Context("when no foundations are given", func() {
			It("returns an error and emits an event", func() {
				environment.Foundations = nil

				event = S.Event{
					Type: "validate.foundationsUnavailable",
					Data: S.PrecheckerEventData{
						Environment: environment,
						Description: "no foundations configured",
					},
				}
				eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)

				err := prechecker.AssertAllFoundationsUp(environment)
				Expect(err).To(MatchError(NoFoundationsConfiguredError{}))

				Expect(eventManager.EmitCall.Received.Events[0]).To(Equal(event))
			})
		})

		Context("when the client returns an error", func() {
			It("returns an error and emits an event", func() {
				environment.Foundations = []string{"bork"}

				event = S.Event{
					Type: "validate.foundationsUnavailable",
					Data: S.PrecheckerEventData{
						Environment: environment,
						Description: "no foundations configured",
					},
				}
				eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)

				err := prechecker.AssertAllFoundationsUp(environment)

				Expect(err.Error()).To(ContainSubstring(InvalidGetRequestError{"bork", errors.New("")}.Error()))
			})
		})

		Context("when all foundations return a 200 OK", func() {
			It("returns a nil error", func() {
				httpStatus = http.StatusOK

				Expect(prechecker.AssertAllFoundationsUp(environment)).To(Succeed())

				Expect(foundationURls).To(ConsistOf("/v2/info"))
			})
		})

		Context("when a foundation returns a 500 internal server error", func() {
			It("returns an error and emits an event", func() {
				event = S.Event{
					Type: "validate.foundationsUnavailable",
					Data: S.PrecheckerEventData{
						Environment: environment,
						Description: "deploy aborted: one or more CF foundations unavailable",
					},
				}
				eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)

				httpStatus = http.StatusInternalServerError

				Expect(prechecker.AssertAllFoundationsUp(environment)).ToNot(Succeed())

				Expect(foundationURls).To(ConsistOf("/v2/info"))
				Expect(eventManager.EmitCall.Received.Events[0]).ToNot(BeNil())
			})
		})

		Context("when a foundation returns a 404 not found", func() {
			It("returns an error and emits an event", func() {
				event = S.Event{
					Type: "validate.foundationsUnavailable",
					Data: S.PrecheckerEventData{
						Environment: environment,
						Description: "deploy aborted: one or more CF foundations unavailable: http://127.0.0.1:51844: 404 Not Found",
					},
				}
				eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)

				httpStatus = http.StatusNotFound

				Expect(prechecker.AssertAllFoundationsUp(environment)).ToNot(Succeed())

				Expect(eventManager.EmitCall.Received.Events[0]).ToNot(BeNil())
			})
		})
	})
})
