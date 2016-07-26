package deployer_test

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/compozed/deployadactyl/config"
	. "github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/compozed/deployadactyl/test/mocks"
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

		blueGreener  *mocks.BlueGreener
		fetcher      *mocks.Fetcher
		prechecker   *mocks.Prechecker
		eventManager *mocks.EventManager

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

		foundations = []string{randomizer.StringRunes(10)}
		buffer = &bytes.Buffer{}

		environments = map[string]config.Environment{}
		environments[environmentName] = config.Environment{
			Name:        environmentName,
			Domain:      domain,
			Foundations: foundations,
		}
	})

	JustBeforeEach(func() {
		deployer = Deployer{blueGreener, environments, fetcher, log, prechecker, eventManager}
	})

	Describe("Deploy", func() {
		Context("with no environments", func() {
			It("returns an error", func() {
				event.Type = "deploy.error"
				errorMessage := "environment not found: " + environmentName

				environments = nil
				deployer = Deployer{blueGreener, environments, fetcher, log, prechecker, eventManager}

				eventManager.EmitCall.Returns.Error = nil

				err := deployer.Deploy(deploymentInfo, buffer)
				Expect(buffer).To(ContainSubstring(errorMessage))
				Expect(err).To(MatchError(errorMessage))

				fmt.Fprint(deployEventData.Writer, buffer.String())
				Expect(eventManager.EmitCall.Received.Event).To(Equal(event))
			})
		})

		Context("when prechecker fails", func() {
			It("returns an error", func() {
				prechecker.AssertAllFoundationsUpCall.Returns.Error = errors.New(deployAborted)

				err := deployer.Deploy(deploymentInfo, buffer)
				Expect(err).To(MatchError("Deploy aborted, one or more CF foundations unavailable"))

				Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environmentName]))
			})
		})

		Context("when fetcher fails", func() {
			It("returns an error", func() {
				prechecker.AssertAllFoundationsUpCall.Returns.Error = nil

				fetcher.FetchCall.Returns.Error = errors.New("bork")
				fetcher.FetchCall.Returns.AppPath = appPath

				err := deployer.Deploy(deploymentInfo, buffer)
				Expect(err).To(HaveOccurred())
				Expect(buffer).To(ContainSubstring("bork"))

				Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environmentName]))
				Expect(fetcher.FetchCall.Received.URL).To(Equal(artifactURL))
				Expect(fetcher.FetchCall.Received.Manifest).To(BeEmpty())
			})
		})

		Describe("bluegreener", func() {
			Context("when all applications start correctly", func() {
				It("is successful", func() {
					event.Type = "deploy.success"

					eventManager.EmitCall.Returns.Error = nil
					fetcher.FetchCall.Returns.Error = nil
					fetcher.FetchCall.Returns.AppPath = appPath
					blueGreener.PushCall.Returns.Error = nil
					prechecker.AssertAllFoundationsUpCall.Returns.Error = nil

					err := deployer.Deploy(deploymentInfo, buffer)
					Expect(err).ToNot(HaveOccurred())

					Expect(buffer).To(ContainSubstring("deploy was successful"))

					fmt.Fprint(deployEventData.Writer, buffer.String())
					Expect(eventManager.EmitCall.Received.Event).To(Equal(event))
					Expect(fetcher.FetchCall.Received.URL).To(Equal(artifactURL))
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
				event.Type = "deploy.failure"

				eventManager.EmitCall.Returns.Error = errors.New("bork")
				prechecker.AssertAllFoundationsUpCall.Returns.Error = nil
				fetcher.FetchCall.Returns.Error = nil
				fetcher.FetchCall.Returns.AppPath = appPath
				blueGreener.PushCall.Returns.Error = errors.New("bork")

				err := deployer.Deploy(deploymentInfo, buffer)
				Expect(err).To(HaveOccurred())

				fmt.Fprint(deployEventData.Writer, buffer.String())
				Expect(eventManager.EmitCall.Received.Event).To(Equal(event))
				Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environmentName]))
				Expect(fetcher.FetchCall.Received.URL).To(Equal(artifactURL))
				Expect(fetcher.FetchCall.Received.Manifest).To(BeEmpty())
				Expect(blueGreener.PushCall.Received.Environment).To(Equal(environments[environmentName]))
				Expect(blueGreener.PushCall.Received.AppPath).To(Equal(appPath))
				Expect(blueGreener.PushCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
			})
		})

		Context("when eventmanager fails", func() {
			It("returns an error", func() {
				event.Type = "deploy.success"

				eventManager.EmitCall.Returns.Error = errors.New("bork")
				prechecker.AssertAllFoundationsUpCall.Returns.Error = nil
				fetcher.FetchCall.Returns.Error = nil
				fetcher.FetchCall.Returns.AppPath = appPath
				blueGreener.PushCall.Returns.Error = nil

				err := deployer.Deploy(deploymentInfo, buffer)
				Expect(err).To(HaveOccurred())

				Expect(buffer).To(ContainSubstring("bork"))

				fmt.Fprint(deployEventData.Writer, buffer.String())
				Expect(eventManager.EmitCall.Received.Event).To(Equal(event))
				Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environmentName]))
				Expect(fetcher.FetchCall.Received.URL).To(Equal(artifactURL))
				Expect(fetcher.FetchCall.Received.Manifest).To(BeEmpty())
				Expect(blueGreener.PushCall.Received.Environment).To(Equal(environments[environmentName]))
				Expect(blueGreener.PushCall.Received.AppPath).To(Equal(appPath))
				Expect(blueGreener.PushCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
			})
		})
	})
})
