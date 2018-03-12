package startstopper_test

/*
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

var _ = Describe("Stopper", func() {
	var (
		starter      startstopper.Starter
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

		starter = startstopper.Starter{
			Courier:       courier,
			CFContext:     cfContext,
			Authorization: auth,
			EventManager:  eventManager,
			Response:      response,
			Log:           logger.DefaultLogger(logBuffer, logging.DEBUG, "pusher_test"),
			FoundationURL: randomFoundationURL,
			AppName :      randomAppName,
		}
	})

	Describe("starting an app", func() {
		Context("when the start succeeds", func() {
			It("returns with success", func() {
				courier.ExistsCall.Returns.Bool = true
				courier.StartCall.Returns.Output = []byte("start succeeded")

				Expect(starter.Execute()).To(Succeed())

				Expect(courier.StartCall.Received.AppName).To(Equal(randomAppName))

				Eventually(response).Should(Say("start succeeded"))

				Eventually(logBuffer).Should(Say(fmt.Sprintf("starting app %s", randomAppName)))
				Eventually(logBuffer).Should(Say(fmt.Sprintf("successfully started app %s", randomAppName)))
			})

			It("emits a StartFinished event", func() {
				courier.ExistsCall.Returns.Bool = true
				courier.StartCall.Returns.Output = []byte("start succeeded")

				Expect(starter.Execute()).To(Succeed())
				Expect(eventManager.EmitCall.Received.Events[0].Type).To(Equal(C.StartFinishedEvent))
			})
		})

		Context("when the start fails", func() {
			It("returns an error", func() {
				courier.ExistsCall.Returns.Bool = true
				courier.StartCall.Returns.Error = errors.New("start error")

				err := starter.Execute()

				Expect(err).To(MatchError(startstopper.StartError{ApplicationName: randomAppName, Out: nil}))
			})
		})

		Context("when the app does not exist", func() {
			It("returns an error", func() {
				courier.ExistsCall.Returns.Bool = false

				err := starter.Execute()

				Expect(err).To(MatchError(startstopper.ExistsError{ApplicationName: randomAppName}))
			})
		})
	})
})
*/
