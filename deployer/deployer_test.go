package deployer_test

import (
	"bytes"

	"github.com/compozed/deployadactyl/config"
	. "github.com/compozed/deployadactyl/deployer"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/compozed/deployadactyl/test/mocks"
	"github.com/go-errors/errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/op/go-logging"
	"github.com/stretchr/testify/mock"
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

		deploymentInfo S.DeploymentInfo
		event          S.Event
		foundations    []string
		environments   = map[string]config.Environment{}
		log            = logger.DefaultLogger(GinkgoWriter, logging.DEBUG, "test")
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

		event = S.Event{
			Data: S.DeployEventData{
				Writer:         &bytes.Buffer{},
				DeploymentInfo: &deploymentInfo,
			},
		}

		foundations = []string{randomizer.StringRunes(10)}
		buffer = &bytes.Buffer{}
	})

	JustBeforeEach(func() {
		deployer = Deployer{blueGreener, environments, fetcher, log, prechecker, eventManager}
	})

	AfterEach(func() {
		Expect(blueGreener.AssertExpectations(GinkgoT())).To(BeTrue())
		Expect(fetcher.AssertExpectations(GinkgoT())).To(BeTrue())
		Expect(prechecker.AssertExpectations(GinkgoT())).To(BeTrue())
		Expect(eventManager.AssertExpectations(GinkgoT())).To(BeTrue())
	})

	Describe("Deploy", func() {
		Context("with no environments", func() {
			It("returns an error", func() {
				err := deployer.Deploy(deploymentInfo, buffer)

				errorMessage := "environment not found: " + environmentName
				Expect(buffer).To(ContainSubstring(errorMessage))
				Expect(err).To(MatchError(errorMessage))
			})
		})

		Context("with at least one bad foundation", func() {
			BeforeEach(func() {
				environments[environmentName] = config.Environment{
					Name:        environmentName,
					Foundations: foundations,
				}

				prechecker.On("AssertAllFoundationsUp", environments[environmentName]).
					Return(errors.New(deployAborted))
			})

			It("returns an error message", func() {
				err := deployer.Deploy(deploymentInfo, buffer)

				Expect(err).To(MatchError("Deploy aborted, one or more CF foundations unavailable"))
			})
		})

		Context("when apps start correctly", func() {
			BeforeEach(func() {
				environments[environmentName] = config.Environment{
					Name:        environmentName,
					Domain:      domain,
					Foundations: foundations,
				}

				fetcher.On("Fetch", artifactURL, "").Return(appPath, nil)

				blueGreener.On("Push",
					environments[environmentName],
					appPath,
					deploymentInfo,
					buffer,
				).Return(nil)

				prechecker.On("AssertAllFoundationsUp", environments[environmentName]).Return(nil)
			})

			It("emits deploy.finish event", func() {
				event.Type = "deploy.finish"
				// using `event` in this mock call causes an error that the mock
				// isn't correct. we are going to use `mock.Anything` until we
				// replace this mock with a hand written one
				eventManager.On("Emit", mock.Anything).Return(nil).Times(1)

				err := deployer.Deploy(deploymentInfo, buffer)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when an app fails to start", func() {
			BeforeEach(func() {
				environments[environmentName] = config.Environment{
					Name:        environmentName,
					Domain:      domain,
					Foundations: foundations,
				}

				fetcher.On("Fetch", artifactURL, "").Return(appPath, nil)

				blueGreener.On("Push",
					environments[environmentName],
					appPath,
					deploymentInfo,
					buffer,
				).Return(errors.New("bork"))

				prechecker.On("AssertAllFoundationsUp", environments[environmentName]).Return(nil)
			})

			It("emits a deploy.error event", func() {
				event.Type = "deploy.error"
				eventManager.On("Emit", event).Return(errors.New("bork")).Times(1)

				err := deployer.Deploy(deploymentInfo, buffer)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when EventManager emits events", func() {
			var (
				buffer *bytes.Buffer
			)

			BeforeEach(func() {
				environments[environmentName] = config.Environment{
					Name:        environmentName,
					Domain:      domain,
					Foundations: foundations,
				}

				prechecker.On("AssertAllFoundationsUp", environments[environmentName]).Return(nil)
				fetcher.On("Fetch", artifactURL, "").Return(appPath, nil)

				buffer = &bytes.Buffer{}

				blueGreener.On("Push",
					environments[environmentName],
					appPath,
					deploymentInfo,
					buffer,
				).Return(nil)
			})

			It("returns an error on deploy.finish", func() {
				eventFinish := S.Event{
					Data: S.DeployEventData{
						Writer:         buffer,
						DeploymentInfo: &deploymentInfo,
					},
					Type: "deploy.finish",
				}

				eventManager.On("Emit", eventFinish).Return(errors.New("bork"))

				err := deployer.Deploy(deploymentInfo, buffer)
				Expect(err).To(HaveOccurred())
				Expect(buffer).To(ContainSubstring("bork"))
			})
		})

		Context("when fetcher returns an error", func() {
			It("returns an error", func() {
				environments[environmentName] = config.Environment{
					Name:        environmentName,
					Domain:      domain,
					Foundations: foundations,
				}

				prechecker.On("AssertAllFoundationsUp", environments[environmentName]).Return(nil)
				fetcher.On("Fetch", artifactURL, "").Return("", errors.New("bork")).Times(1)

				err := deployer.Deploy(deploymentInfo, buffer)
				Expect(err).To(HaveOccurred())
				Expect(buffer).To(ContainSubstring("bork"))
			})
		})
	})
})
