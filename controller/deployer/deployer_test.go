package deployer_test

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"github.com/compozed/deployadactyl/config"
	. "github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/op/go-logging"
)

const (
	deployAborted = "Deploy aborted, one or more CF foundations unavailable"
)

var _ = Describe("Deployer", func() {
	var (
		deployer Deployer

		c              config.Config
		blueGreener    *mocks.BlueGreener
		fetcher        *mocks.Fetcher
		prechecker     *mocks.Prechecker
		eventManager   *mocks.EventManager
		randomizerMock *mocks.Randomizer

		req             *http.Request
		reqBuffer       *bytes.Buffer
		appName         string
		appPath         string
		artifactURL     string
		domain          string
		environmentName string
		org             string
		space           string
		username        string
		uuid            string
		password        string
		buffer          *bytes.Buffer

		deploymentInfo  S.DeploymentInfo
		event           S.Event
		deployEventData S.DeployEventData
		foundations     []string
		environments    = map[string]config.Environment{}
		log             = logger.DefaultLogger(GinkgoWriter, logging.DEBUG, "test")
	)

	BeforeEach(func() {
		blueGreener = &mocks.BlueGreener{}
		fetcher = &mocks.Fetcher{}
		prechecker = &mocks.Prechecker{}
		eventManager = &mocks.EventManager{}
		randomizerMock = &mocks.Randomizer{}

		appName = "appName-" + randomizer.StringRunes(10)
		appPath = "appPath-" + randomizer.StringRunes(10)
		artifactURL = "artifactURL-" + randomizer.StringRunes(10)
		domain = "domain-" + randomizer.StringRunes(10)
		environmentName = "environmentName-" + randomizer.StringRunes(10)
		org = "org-" + randomizer.StringRunes(10)
		password = "password-" + randomizer.StringRunes(10)
		space = "space-" + randomizer.StringRunes(10)
		username = "username-" + randomizer.StringRunes(10)
		uuid = "uuid-" + randomizer.StringRunes(10)

		randomizerMock.RandomizeCall.Returns.Runes = uuid

		reqBuffer = bytes.NewBufferString(fmt.Sprintf(`{
		  		"artifact_url": "%s"
		  	}`,
			artifactURL,
		))

		req, _ = http.NewRequest("POST", "", reqBuffer)

		deploymentInfo = S.DeploymentInfo{
			ArtifactURL: artifactURL,
			Username:    username,
			Password:    password,
			Environment: environmentName,
			Org:         org,
			Space:       space,
			AppName:     appName,
			UUID:        uuid,
		}

		deployEventData = S.DeployEventData{
			Writer:         &bytes.Buffer{},
			DeploymentInfo: &deploymentInfo,
		}

		event = S.Event{
			Data: deployEventData,
		}

		randomizerMock.RandomizeCall.Returns.Runes = uuid

		foundations = []string{randomizer.StringRunes(10)}
		buffer = &bytes.Buffer{}

		environments = map[string]config.Environment{}
		environments[environmentName] = config.Environment{
			Name:        environmentName,
			Domain:      domain,
			Foundations: foundations,
		}

		c = config.Config{
			Username:     username,
			Password:     password,
			Environments: environments,
		}

		deployer = Deployer{c, blueGreener, fetcher, prechecker, eventManager, randomizerMock, log}
	})

	Describe("Deploy JSON", func() {
		Context("with no environments", func() {
			It("returns an error", func() {
				errorMessage := "environment not found: " + environmentName

				eventManager.EmitCall.Returns.Error = nil

				emptyConfiguration := config.Config{
					Username:     "",
					Password:     "",
					Environments: nil,
				}

				deployer = Deployer{emptyConfiguration, blueGreener, fetcher, prechecker, eventManager, randomizerMock, log}
				err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, buffer)
				Expect(buffer).To(ContainSubstring(errorMessage))
				Expect(err).To(MatchError(errorMessage))
				Expect(statusCode).To(Equal(500))

				fmt.Fprint(deployEventData.Writer, buffer.String())
			})
		})

		Context("when prechecker fails", func() {
			It("returns an error", func() {
				prechecker.AssertAllFoundationsUpCall.Returns.Error = errors.New(deployAborted)

				err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, buffer)
				Expect(err).To(MatchError("Deploy aborted, one or more CF foundations unavailable"))
				Expect(statusCode).To(Equal(500))

				Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environmentName]))
			})
		})

		Context("when fetcher fails", func() {
			It("returns an error", func() {
				prechecker.AssertAllFoundationsUpCall.Returns.Error = nil

				fetcher.FetchCall.Returns.Error = errors.New("Fetcher error")
				fetcher.FetchCall.Returns.AppPath = appPath

				err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, buffer)
				Expect(err).To(MatchError("Fetcher error"))
				Expect(statusCode).To(Equal(500))

				Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environmentName]))
				Expect(fetcher.FetchCall.Received.ArtifactURL).To(Equal(artifactURL))
				Expect(fetcher.FetchCall.Received.Manifest).To(BeEmpty())
			})
		})

		Describe("bluegreener", func() {
			Context("when all applications start correctly", func() {
				It("is successful", func() {
					eventManager.EmitCall.Returns.Error = nil
					fetcher.FetchCall.Returns.Error = nil
					fetcher.FetchCall.Returns.AppPath = appPath
					blueGreener.PushCall.Returns.Error = nil
					prechecker.AssertAllFoundationsUpCall.Returns.Error = nil

					err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, buffer)
					Expect(err).To(BeNil())
					Expect(statusCode).To(Equal(200))
					Expect(buffer).To(ContainSubstring("deploy was successful"))
					fmt.Fprint(deployEventData.Writer, buffer.String())
					Expect(fetcher.FetchCall.Received.ArtifactURL).To(Equal(artifactURL))
					Expect(fetcher.FetchCall.Received.Manifest).To(BeEmpty())
					Expect(blueGreener.PushCall.Received.Environment).To(Equal(environments[environmentName]))
					Expect(blueGreener.PushCall.Received.AppPath).To(Equal(appPath))
					Expect(blueGreener.PushCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
					Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environmentName]))
				})
			})
		})

		Context("when an application fails to start", func() {
			It("returns an error", func() {
				eventManager.EmitCall.Returns.Error = nil
				prechecker.AssertAllFoundationsUpCall.Returns.Error = nil
				fetcher.FetchCall.Returns.Error = nil
				fetcher.FetchCall.Returns.AppPath = appPath

				By("making bluegreener return an error")
				blueGreener.PushCall.Returns.Error = errors.New("blue green error")

				err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, buffer)
				Expect(err).To(MatchError("blue green error"))
				Expect(statusCode).To(Equal(500))

				fmt.Fprint(deployEventData.Writer, buffer.String())
				Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environmentName]))
				Expect(fetcher.FetchCall.Received.ArtifactURL).To(Equal(artifactURL))
				Expect(fetcher.FetchCall.Received.Manifest).To(BeEmpty())
				Expect(blueGreener.PushCall.Received.Environment).To(Equal(environments[environmentName]))
				Expect(blueGreener.PushCall.Received.AppPath).To(Equal(appPath))
				Expect(blueGreener.PushCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
			})
		})

		Context("when eventmanager fails on deploy.start", func() {
			It("returns an error", func() {
				By("making eventmanager return an error")
				eventManager.EmitCall.Returns.Error = errors.New("Event error")

				err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, buffer)
				Expect(err).To(MatchError("an error occurred in the deploy.start event"))
				Expect(statusCode).To(Equal(500))
				Expect(buffer).To(ContainSubstring("Event error"))

				fmt.Fprint(deployEventData.Writer, buffer.String())
			})
		})
	})

	Describe("Deploy Zip", func() {
		Context("when all applications start correctly", func() {
			It("is successful", func() {

			})
		})
		Context("when authentication is required", func() {
			It("sets authentication from authentication header", func() {

			})

			It("sets the authentication from the config when there is no authentication header", func() {

			})
		})
		Context("when manifest file cannot be found in the extracted zip", func() {
			It("returns an error and status code 400", func() {

			})
		})
		Context("when the environment cannot be found", func() {
			It("returns an error", func() {

			})
		})
		Context("prechecker fails", func() {
			It("returns an error", func() {

			})
		})
		Context("push fails", func() {
			It("returns an error", func() {

			})
		})
		Context("deploy event handler fails", func() {
			It("returns an error", func() {

			})
		})
	})
})
