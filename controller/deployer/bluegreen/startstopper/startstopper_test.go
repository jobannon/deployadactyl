package startstopper_test

import (
	"errors"
	//"fmt"
	"math/rand"

	C "github.com/compozed/deployadactyl/constants"
	. "github.com/compozed/deployadactyl/controller/deployer/bluegreen/pusher"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/op/go-logging"

	"fmt"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen/startstopper"
	"github.com/compozed/deployadactyl/interfaces"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("StartStopper", func() {
	var (
		startStopper startstopper.StartStopper
		courier      *mocks.Courier
		eventManager *mocks.EventManager

		randomUsername      string
		randomPassword      string
		randomOrg           string
		randomSpace         string
		randomDomain        string
		randomAppPath       string
		randomAppName       string
		randomInstances     uint16
		randomUUID          string
		randomEndpoint      string
		randomFoundationURL string
		tempAppWithUUID     string
		skipSSL             bool
		deploymentInfo      S.DeploymentInfo
		cfContext           interfaces.CFContext
		auth                interfaces.Authorization
		response            *Buffer
		logBuffer           *Buffer
	)

	BeforeEach(func() {
		courier = &mocks.Courier{}
		eventManager = &mocks.EventManager{}

		randomFoundationURL = "randomFoundationURL-" + randomizer.StringRunes(10)
		randomUsername = "randomUsername-" + randomizer.StringRunes(10)
		randomPassword = "randomPassword-" + randomizer.StringRunes(10)
		randomOrg = "randomOrg-" + randomizer.StringRunes(10)
		randomSpace = "randomSpace-" + randomizer.StringRunes(10)
		randomDomain = "randomDomain-" + randomizer.StringRunes(10)
		randomAppPath = "randomAppPath-" + randomizer.StringRunes(10)
		randomAppName = "randomAppName-" + randomizer.StringRunes(10)
		randomEndpoint = "randomEndpoint-" + randomizer.StringRunes(10)
		randomUUID = randomizer.StringRunes(10)
		randomInstances = uint16(rand.Uint32())

		tempAppWithUUID = randomAppName + TemporaryNameSuffix + randomUUID

		response = NewBuffer()
		logBuffer = NewBuffer()

		eventManager.EmitCall.Returns.Error = append(eventManager.EmitCall.Returns.Error, nil)

		deploymentInfo = S.DeploymentInfo{
			Username:            randomUsername,
			Password:            randomPassword,
			Org:                 randomOrg,
			Space:               randomSpace,
			AppName:             randomAppName,
			SkipSSL:             skipSSL,
			Instances:           randomInstances,
			Domain:              randomDomain,
			UUID:                randomUUID,
			HealthCheckEndpoint: randomEndpoint,
		}

		cfContext = interfaces.CFContext{
			Organization: randomOrg,
			Space:        randomSpace,
			Application:  randomAppName,
		}

		auth = interfaces.Authorization{
			Username: randomUsername,
			Password: randomPassword,
		}

		startStopper = startstopper.StartStopper{
			Courier:       courier,
			CFContext:     cfContext,
			Authorization: auth,
			EventManager:  eventManager,
			Response:      response,
			Log:           logger.DefaultLogger(logBuffer, logging.DEBUG, "pusher_test"),
		}
	})

	Describe("logging in", func() {
		Context("when login succeeds", func() {
			It("gives the correct info to the courier", func() {

				Expect(startStopper.Login(randomFoundationURL)).To(Succeed())

				Expect(courier.LoginCall.Received.FoundationURL).To(Equal(randomFoundationURL))
				Expect(courier.LoginCall.Received.Username).To(Equal(randomUsername))
				Expect(courier.LoginCall.Received.Password).To(Equal(randomPassword))
				Expect(courier.LoginCall.Received.Org).To(Equal(randomOrg))
				Expect(courier.LoginCall.Received.Space).To(Equal(randomSpace))
				Expect(courier.LoginCall.Received.SkipSSL).To(Equal(skipSSL))
			})

			It("writes the output of the courier to the response", func() {
				courier.LoginCall.Returns.Output = []byte("login succeeded")

				Expect(startStopper.Login(randomFoundationURL)).To(Succeed())

				Eventually(response).Should(Say("login succeeded"))
			})
		})

		Context("when login fails", func() {
			It("returns an error", func() {
				courier.LoginCall.Returns.Output = []byte("login output")
				courier.LoginCall.Returns.Error = errors.New("login error")

				err := startStopper.Login(randomFoundationURL)
				Expect(err).To(MatchError(LoginError{randomFoundationURL, []byte("login output")}))
			})

			It("writes the output of the courier to the response", func() {
				courier.LoginCall.Returns.Output = []byte("login output")
				courier.LoginCall.Returns.Error = errors.New("login error")

				err := startStopper.Login(randomFoundationURL)
				Expect(err).To(HaveOccurred())

				Eventually(response).Should(Say("login output"))
			})

			It("logs an error", func() {
				courier.LoginCall.Returns.Error = errors.New("login error")

				err := startStopper.Login(randomFoundationURL)
				Expect(err).To(HaveOccurred())

				Eventually(logBuffer).Should(Say(fmt.Sprintf("could not login to %s", randomFoundationURL)))
			})
		})
	})

	Describe("stopping an app", func() {
		Context("when the stop succeeds", func() {
			It("returns with success", func() {
				courier.ExistsCall.Returns.Bool = true
				courier.StopCall.Returns.Output = []byte("stop succeeded")

				Expect(startStopper.Stop(randomAppName, randomFoundationURL)).To(Succeed())

				Expect(courier.StopCall.Received.AppName).To(Equal(randomAppName))

				Eventually(response).Should(Say("stop succeeded"))

				Eventually(logBuffer).Should(Say(fmt.Sprintf("stopping app %s", randomAppName)))
				Eventually(logBuffer).Should(Say(fmt.Sprintf("successfully stopped app %s", randomAppName)))
			})

			It("emits a StopFinished event", func() {
				courier.ExistsCall.Returns.Bool = true
				courier.StopCall.Returns.Output = []byte("stop succeeded")

				Expect(startStopper.Stop(randomAppName, randomFoundationURL)).To(Succeed())
				Expect(eventManager.EmitCall.Received.Events[0].Type).To(Equal(C.StopFinishedEvent))
			})
		})

		Context("when the stop fails", func() {
			It("returns an error", func() {
				courier.ExistsCall.Returns.Bool = true
				courier.StopCall.Returns.Error = errors.New("stop error")

				err := startStopper.Stop(randomAppName, randomFoundationURL)

				Expect(err).To(MatchError(startstopper.StopError{ApplicationName: randomAppName, Out: nil}))
			})
		})

		Context("when the app does not exist", func() {
			It("returns an error", func() {
				courier.ExistsCall.Returns.Bool = false

				err := startStopper.Stop(randomAppName, randomFoundationURL)

				Expect(err).To(MatchError(startstopper.ExistsError{ApplicationName: randomAppName}))
			})
		})
	})

	Describe("starting an app", func() {
		Context("when the start succeeds", func() {
			It("returns with success", func() {
				courier.ExistsCall.Returns.Bool = true
				courier.StartCall.Returns.Output = []byte("start succeeded")

				Expect(startStopper.Start(randomAppName, randomFoundationURL)).To(Succeed())

				Expect(courier.StartCall.Received.AppName).To(Equal(randomAppName))

				Eventually(response).Should(Say("start succeeded"))

				Eventually(logBuffer).Should(Say(fmt.Sprintf("starting app %s", randomAppName)))
				Eventually(logBuffer).Should(Say(fmt.Sprintf("successfully started app %s", randomAppName)))
			})

			It("emits a StartFinished event", func() {
				courier.ExistsCall.Returns.Bool = true
				courier.StartCall.Returns.Output = []byte("start succeeded")

				Expect(startStopper.Start(randomAppName, randomFoundationURL)).To(Succeed())
				Expect(eventManager.EmitCall.Received.Events[0].Type).To(Equal(C.StartFinishedEvent))
			})
		})

		Context("when the start fails", func() {
			It("returns an error", func() {
				courier.ExistsCall.Returns.Bool = true
				courier.StartCall.Returns.Error = errors.New("start error")

				err := startStopper.Start(randomAppName, randomFoundationURL)

				Expect(err).To(MatchError(startstopper.StartError{ApplicationName: randomAppName, Out: nil}))
			})
		})

		Context("when the app does not exist", func() {
			It("returns an error", func() {
				courier.ExistsCall.Returns.Bool = false

				err := startStopper.Start(randomAppName, randomFoundationURL)

				Expect(err).To(MatchError(startstopper.ExistsError{ApplicationName: randomAppName}))
			})
		})
	})

	//
	//	Context("when the application does not exist", func() {
	//		It("does not delete the non-existant original application", func() {
	//			courier.ExistsCall.Returns.Bool = false
	//
	//			err := pusher.FinishPush()
	//			Expect(err).ToNot(HaveOccurred())
	//
	//			Expect(courier.DeleteCall.Received.AppName).To(BeEmpty())
	//
	//			Eventually(logBuffer).ShouldNot(Say("delete"))
	//		})
	//	})
	//})
	//
	//Describe("undoing a push", func() {
	//	Context("when the app exists", func() {
	//		BeforeEach(func() {
	//			courier.ExistsCall.Returns.Bool = true
	//		})
	//
	//		It("check that the app exists", func() {
	//			Expect(pusher.UndoPush()).To(Succeed())
	//			Expect(courier.ExistsCall.Received.AppName).To(Equal(randomAppName))
	//		})
	//
	//		It("deletes the app that was pushed", func() {
	//			Expect(pusher.UndoPush()).To(Succeed())
	//
	//			Expect(courier.DeleteCall.Received.AppName).To(Equal(randomAppName + TemporaryNameSuffix + randomUUID))
	//
	//			Eventually(logBuffer).Should(Say(fmt.Sprintf("rolling back deploy of %s", randomAppName)))
	//			Eventually(logBuffer).Should(Say(fmt.Sprintf("deleted %s", randomAppName)))
	//		})
	//
	//		Context("when deleting fails", func() {
	//			It("returns an error and writes a message to the info log", func() {
	//				courier.DeleteCall.Returns.Output = []byte("delete call output")
	//				courier.DeleteCall.Returns.Error = errors.New("delete error")
	//
	//				err := pusher.UndoPush()
	//				Expect(err).To(MatchError(DeleteApplicationError{tempAppWithUUID, []byte("delete call output")}))
	//
	//				Eventually(logBuffer).Should(Say(fmt.Sprintf("could not delete %s", tempAppWithUUID)))
	//			})
	//		})
	//	})
	//
	//	Context("when the app does not exist", func() {
	//		It("renames the newly built app to the intended application name", func() {
	//			Expect(pusher.UndoPush()).To(Succeed())
	//
	//			Expect(courier.RenameCall.Received.AppName).To(Equal(randomAppName + TemporaryNameSuffix + randomUUID))
	//			Expect(courier.RenameCall.Received.AppNameVenerable).To(Equal(randomAppName))
	//
	//			Eventually(logBuffer).Should(Say("renamed %s to %s", tempAppWithUUID, randomAppName))
	//		})
	//
	//		Context("when renaming fails", func() {
	//			It("returns an error and writes a message to the info log", func() {
	//				courier.RenameCall.Returns.Error = errors.New("rename error")
	//				courier.RenameCall.Returns.Output = []byte("rename error")
	//
	//				err := pusher.UndoPush()
	//				Expect(err).To(MatchError(RenameError{tempAppWithUUID, []byte("rename error")}))
	//
	//				Eventually(logBuffer).Should(Say(fmt.Sprintf("could not rename %s to %s", tempAppWithUUID, randomAppName)))
	//			})
	//		})
	//	})
	//})
	//
	//Describe("cleaning up temporary directories", func() {
	//	It("is successful", func() {
	//		courier.CleanUpCall.Returns.Error = nil
	//
	//		Expect(pusher.CleanUp()).To(Succeed())
	//	})
	//})
	//
	//Describe("event handling", func() {
	//	Context("when a PushFinishedEvent is emitted", func() {
	//		It("does not return an error", func() {
	//			Expect(pusher.Push(randomAppPath, randomFoundationURL)).To(Succeed())
	//
	//			Expect(eventManager.EmitCall.Received.Events[0].Type).To(Equal(C.PushFinishedEvent))
	//		})
	//
	//		It("has the temporary app name on the event", func() {
	//			Expect(pusher.Push(randomAppPath, randomFoundationURL)).To(Succeed())
	//
	//			Expect(eventManager.EmitCall.Received.Events[0].Data.(S.PushEventData).TempAppWithUUID).To(Equal(randomAppName + TemporaryNameSuffix + randomUUID))
	//		})
	//	})
	//
	//	Context("when an event fails", func() {
	//		It("returns an error", func() {
	//			eventManager.EmitCall.Returns.Error[0] = errors.New("event manager error")
	//
	//			err := pusher.Push(randomAppPath, randomFoundationURL)
	//			Expect(err).To(MatchError("event manager error"))
	//
	//			Expect(eventManager.EmitCall.Received.Events[0].Type).To(Equal(C.PushFinishedEvent))
	//		})
	//	})
	//})
})
