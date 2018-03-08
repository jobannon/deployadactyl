package statemanager_test

import (
	"math/rand"

	"github.com/compozed/deployadactyl/config"
	. "github.com/compozed/deployadactyl/controller/statemanager"

	. "github.com/onsi/ginkgo"

	"bytes"
	"errors"
	"github.com/compozed/deployadactyl/constants"
	"github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/compozed/deployadactyl/structs"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/op/go-logging"
	"net/http"
)

var _ = Describe("StateManager", func() {
	Context("when Stop is called", func() {
		It("fails if the environment does not exist", func() {
			environment := "environment-" + randomizer.StringRunes(10)
			context := interfaces.CFContext{
				Environment:  environment,
				Organization: "org-" + randomizer.StringRunes(10),
				Space:        "space-" + randomizer.StringRunes(10),
				Application:  "appName-" + randomizer.StringRunes(10),
			}
			uuid := "uuid-" + randomizer.StringRunes(10)
			response := &bytes.Buffer{}
			environments := map[string]structs.Environment{}

			c := config.Config{
				Username:     "username-" + randomizer.StringRunes(10),
				Password:     "password-" + randomizer.StringRunes(10),
				Environments: environments,
			}

			manager := StateManager{
				Config: c,
			}

			auth := interfaces.Authorization{
				Username: "authuser",
				Password: "authpassword",
			}

			statusCode, _, err := manager.Stop(context, uuid, auth, response)
			Expect(err).To(HaveOccurred())
			Expect(statusCode).To(Equal(http.StatusInternalServerError))
		})

		It("checks that all foundations are alive", func() {
			environment := "environment-" + randomizer.StringRunes(10)
			uuid := "uuid-" + randomizer.StringRunes(10)
			context := interfaces.CFContext{
				Environment:  environment,
				Organization: "org-" + randomizer.StringRunes(10),
				Space:        "space-" + randomizer.StringRunes(10),
				Application:  "appName-" + randomizer.StringRunes(10),
			}
			response := &bytes.Buffer{}
			prechecker := &mocks.Prechecker{}
			environments := map[string]structs.Environment{}
			environments[environment] = structs.Environment{
				Name:           environment,
				Domain:         "domain-" + randomizer.StringRunes(10),
				Foundations:    []string{randomizer.StringRunes(10)},
				Instances:      uint16(rand.Uint32()),
				CustomParams:   make(map[string]interface{}),
				EnableRollback: true,
			}

			c := config.Config{
				Username:     "username-" + randomizer.StringRunes(10),
				Password:     "password-" + randomizer.StringRunes(10),
				Environments: environments,
			}

			manager := StateManager{
				Prechecker:   prechecker,
				Config:       c,
				Log:          logger.DefaultLogger(NewBuffer(), logging.DEBUG, "state manager tests"),
				EventManager: &mocks.EventManager{},
				BlueGreener:  &mocks.BlueGreener{},
			}

			auth := interfaces.Authorization{
				Username: "authuser",
				Password: "authpassword",
			}

			manager.Stop(context, uuid, auth, response)

			Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environment]))
		})

		It("fails if one or more foundations are inactive", func() {
			environment := "environment-" + randomizer.StringRunes(10)
			context := interfaces.CFContext{
				Environment:  environment,
				Organization: "org-" + randomizer.StringRunes(10),
				Space:        "space-" + randomizer.StringRunes(10),
				Application:  "appName-" + randomizer.StringRunes(10),
			}
			uuid := "uuid-" + randomizer.StringRunes(10)
			logBuffer := NewBuffer()
			response := &bytes.Buffer{}
			prechecker := &mocks.Prechecker{}
			prechecker.AssertAllFoundationsUpCall.Returns.Error = errors.New("error occurred")

			environments := map[string]structs.Environment{}
			environments[environment] = structs.Environment{
				Name:           environment,
				Domain:         "domain-" + randomizer.StringRunes(10),
				Foundations:    []string{randomizer.StringRunes(10)},
				Instances:      uint16(rand.Uint32()),
				CustomParams:   make(map[string]interface{}),
				EnableRollback: true,
			}

			c := config.Config{
				Username:     "username-" + randomizer.StringRunes(10),
				Password:     "password-" + randomizer.StringRunes(10),
				Environments: environments,
			}

			manager := StateManager{
				Prechecker: prechecker,
				Config:     c,
				Log:        logger.DefaultLogger(logBuffer, logging.DEBUG, "state manager tests"),
			}

			auth := interfaces.Authorization{
				Username: "authuser",
				Password: "authpassword",
			}

			statusCode, _, err := manager.Stop(context, uuid, auth, response)
			Expect(statusCode).To(Equal(http.StatusInternalServerError))
			Expect(err.Error()).To(Equal("error occurred"))
		})

		It("returns correct deployment info", func() {
			environment := "environment-" + randomizer.StringRunes(10)
			context := interfaces.CFContext{
				Environment:  environment,
				Organization: "org-" + randomizer.StringRunes(10),
				Space:        "space-" + randomizer.StringRunes(10),
				Application:  "appName-" + randomizer.StringRunes(10),
			}
			uuid := "uuid-" + randomizer.StringRunes(10)
			response := &bytes.Buffer{}
			prechecker := &mocks.Prechecker{}
			environments := map[string]structs.Environment{}
			environments[environment] = structs.Environment{
				Name:           environment,
				Domain:         "domain-" + randomizer.StringRunes(10),
				Foundations:    []string{randomizer.StringRunes(10)},
				Instances:      uint16(rand.Uint32()),
				CustomParams:   make(map[string]interface{}),
				EnableRollback: true,
			}

			c := config.Config{
				Username:     "username-" + randomizer.StringRunes(10),
				Password:     "password-" + randomizer.StringRunes(10),
				Environments: environments,
			}

			manager := StateManager{
				Prechecker:   prechecker,
				Config:       c,
				Log:          logger.DefaultLogger(NewBuffer(), logging.DEBUG, "state manager tests"),
				EventManager: &mocks.EventManager{},
				BlueGreener:  &mocks.BlueGreener{},
			}

			auth := interfaces.Authorization{
				Username: "authuser",
				Password: "authpassword",
			}

			statusCode, deploymentInfo, err := manager.Stop(context, uuid, auth, response)
			Expect(err).ToNot(HaveOccurred())
			Expect(statusCode).To(Equal(http.StatusOK))
			Expect(deploymentInfo.Username).To(Equal(auth.Username))
			Expect(deploymentInfo.Password).To(Equal(auth.Password))
			Expect(deploymentInfo.Environment).To(Equal(environment))
			Expect(deploymentInfo.Org).To(Equal(context.Organization))
			Expect(deploymentInfo.Space).To(Equal(context.Space))
			Expect(deploymentInfo.AppName).To(Equal(context.Application))
			Expect(deploymentInfo.UUID).To(Equal(uuid))
			Expect(deploymentInfo.SkipSSL).To(Equal(environments[environment].SkipSSL))
		})

		It("logs context info", func() {
			environment := "environment-" + randomizer.StringRunes(10)
			context := interfaces.CFContext{
				Environment:  environment,
				Organization: "org-" + randomizer.StringRunes(10),
				Space:        "space-" + randomizer.StringRunes(10),
				Application:  "appName-" + randomizer.StringRunes(10),
			}
			uuid := "uuid-" + randomizer.StringRunes(10)
			response := NewBuffer()
			prechecker := &mocks.Prechecker{}
			environments := map[string]structs.Environment{}
			environments[environment] = structs.Environment{
				Name:           environment,
				Domain:         "domain-" + randomizer.StringRunes(10),
				Foundations:    []string{randomizer.StringRunes(10)},
				Instances:      uint16(rand.Uint32()),
				CustomParams:   make(map[string]interface{}),
				EnableRollback: true,
			}

			c := config.Config{
				Username:     "username-" + randomizer.StringRunes(10),
				Password:     "password-" + randomizer.StringRunes(10),
				Environments: environments,
			}

			eventManager := &mocks.EventManager{}

			manager := StateManager{
				Prechecker:   prechecker,
				Config:       c,
				Log:          logger.DefaultLogger(NewBuffer(), logging.DEBUG, "state manager tests"),
				EventManager: eventManager,
				BlueGreener:  &mocks.BlueGreener{},
			}

			auth := interfaces.Authorization{
				Username: "authuser",
				Password: "authpassword",
			}

			manager.Stop(context, uuid, auth, response)

			Expect(string(response.Contents())).To(ContainSubstring(environment))
			Expect(string(response.Contents())).To(ContainSubstring(auth.Username))
			Expect(string(response.Contents())).To(ContainSubstring(context.Organization))
			Expect(string(response.Contents())).To(ContainSubstring(context.Space))
			Expect(string(response.Contents())).To(ContainSubstring(context.Application))
		})

		It("Emits Success Event", func() {
			environment := "environment-" + randomizer.StringRunes(10)
			context := interfaces.CFContext{
				Environment:  environment,
				Organization: "org-" + randomizer.StringRunes(10),
				Space:        "space-" + randomizer.StringRunes(10),
				Application:  "appName-" + randomizer.StringRunes(10),
			}
			uuid := "uuid-" + randomizer.StringRunes(10)
			response := NewBuffer()
			prechecker := &mocks.Prechecker{}
			environments := map[string]structs.Environment{}
			environments[environment] = structs.Environment{
				Name:           environment,
				Domain:         "domain-" + randomizer.StringRunes(10),
				Foundations:    []string{randomizer.StringRunes(10)},
				Instances:      uint16(rand.Uint32()),
				CustomParams:   make(map[string]interface{}),
				EnableRollback: true,
			}

			c := config.Config{
				Username:     "username-" + randomizer.StringRunes(10),
				Password:     "password-" + randomizer.StringRunes(10),
				Environments: environments,
			}

			eventManager := &mocks.EventManager{}

			manager := StateManager{
				Prechecker:   prechecker,
				Config:       c,
				Log:          logger.DefaultLogger(NewBuffer(), logging.DEBUG, "state manager tests"),
				EventManager: eventManager,
				BlueGreener:  &mocks.BlueGreener{},
			}

			auth := interfaces.Authorization{
				Username: "authuser",
				Password: "authpassword",
			}

			manager.Stop(context, uuid, auth, response)

			Expect(eventManager.EmitCall.Received.Events[0].Type).To(Equal(constants.StopStartEvent))
			Expect(eventManager.EmitCall.Received.Events[0].Data).ToNot(BeNil())
			Expect(eventManager.EmitCall.Received.Events[1].Type).To(Equal(constants.StopSuccessEvent))
			Expect(eventManager.EmitCall.Received.Events[1].Data).ToNot(BeNil())
		})

		It("emits stop start event", func() {
			environment := "environment-" + randomizer.StringRunes(10)
			context := interfaces.CFContext{
				Environment:  environment,
				Organization: "org-" + randomizer.StringRunes(10),
				Space:        "space-" + randomizer.StringRunes(10),
				Application:  "appName-" + randomizer.StringRunes(10),
			}
			uuid := "uuid-" + randomizer.StringRunes(10)
			response := NewBuffer()
			prechecker := &mocks.Prechecker{}
			environments := map[string]structs.Environment{}
			environments[environment] = structs.Environment{
				Name:           environment,
				Domain:         "domain-" + randomizer.StringRunes(10),
				Foundations:    []string{randomizer.StringRunes(10)},
				Instances:      uint16(rand.Uint32()),
				CustomParams:   make(map[string]interface{}),
				EnableRollback: true,
			}

			c := config.Config{
				Username:     "username-" + randomizer.StringRunes(10),
				Password:     "password-" + randomizer.StringRunes(10),
				Environments: environments,
			}

			eventManager := &mocks.EventManager{}

			manager := StateManager{
				Prechecker:   prechecker,
				Config:       c,
				Log:          logger.DefaultLogger(NewBuffer(), logging.DEBUG, "state manager tests"),
				EventManager: eventManager,
				BlueGreener:  &mocks.BlueGreener{},
			}

			auth := interfaces.Authorization{
				Username: "authuser",
				Password: "authpassword",
			}

			manager.Stop(context, uuid, auth, response)

			Expect(eventManager.EmitCall.Received.Events[0].Type).To(Equal(constants.StopStartEvent))
			Expect(eventManager.EmitCall.Received.Events[0].Data).ToNot(BeNil())
		})
		It("emits stop start event error", func() {
			eventManager := &mocks.EventManager{}
			eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, errors.New(constants.StopStartEvent+" error"))

			environment := "environment-" + randomizer.StringRunes(10)

			context := interfaces.CFContext{
				Environment:  environment,
				Organization: "org-" + randomizer.StringRunes(10),
				Space:        "space-" + randomizer.StringRunes(10),
				Application:  "appName-" + randomizer.StringRunes(10),
			}
			uuid := "uuid-" + randomizer.StringRunes(10)
			response := NewBuffer()
			prechecker := &mocks.Prechecker{}
			environments := map[string]structs.Environment{}
			environments[environment] = structs.Environment{
				Name:           environment,
				Domain:         "domain-" + randomizer.StringRunes(10),
				Foundations:    []string{randomizer.StringRunes(10)},
				Instances:      uint16(rand.Uint32()),
				CustomParams:   make(map[string]interface{}),
				EnableRollback: true,
			}

			c := config.Config{
				Username:     "username-" + randomizer.StringRunes(10),
				Password:     "password-" + randomizer.StringRunes(10),
				Environments: environments,
			}

			manager := StateManager{
				Prechecker:   prechecker,
				Config:       c,
				Log:          logger.DefaultLogger(NewBuffer(), logging.DEBUG, "state manager tests"),
				EventManager: eventManager,
			}

			auth := interfaces.Authorization{
				Username: "authuser",
				Password: "authpassword",
			}

			statusCode, _, err := manager.Stop(context, uuid, auth, response)

			Expect(statusCode).To(Equal(http.StatusInternalServerError))
			Expect(err.Error()).To(Equal("an error occurred in the stop.start event: stop.start error"))
		})
	})
	Describe("BlueGreener.Stop", func() {
		var (
			blueGreener *mocks.BlueGreener
		)
		Context("when BlueGreener fails with a stop failed error", func() {
			It("returns an error", func() {
				environment := "environment-" + randomizer.StringRunes(10)
				context := interfaces.CFContext{
					Environment:  environment,
					Organization: "org-" + randomizer.StringRunes(10),
					Space:        "space-" + randomizer.StringRunes(10),
					Application:  "appName-" + randomizer.StringRunes(10),
				}
				uuid := "uuid-" + randomizer.StringRunes(10)
				response := NewBuffer()
				prechecker := &mocks.Prechecker{}
				environments := map[string]structs.Environment{}
				environments[environment] = structs.Environment{
					Name:           environment,
					Domain:         "domain-" + randomizer.StringRunes(10),
					Foundations:    []string{randomizer.StringRunes(10)},
					Instances:      uint16(rand.Uint32()),
					CustomParams:   make(map[string]interface{}),
					EnableRollback: true,
				}

				c := config.Config{
					Username:     "username-" + randomizer.StringRunes(10),
					Password:     "password-" + randomizer.StringRunes(10),
					Environments: environments,
				}

				eventManager := &mocks.EventManager{}

				blueGreener = &mocks.BlueGreener{}
				manager := StateManager{
					Prechecker:   prechecker,
					Config:       c,
					Log:          logger.DefaultLogger(NewBuffer(), logging.DEBUG, "state manager tests"),
					EventManager: eventManager,
					BlueGreener:  blueGreener,
				}

				auth := interfaces.Authorization{
					Username: "authuser",
					Password: "authpassword",
				}

				expectedError := errors.New("stop failed")
				blueGreener.StopCall.Returns.Error = expectedError

				statusCode, _, err := manager.Stop(context, uuid, auth, response)

				Expect(err.Error()).To(Equal(expectedError.Error()))

				Expect(statusCode).To(Equal(http.StatusInternalServerError))
			})
		})
	})
})
