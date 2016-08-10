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

	Describe("deploy JSON", func() {

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

		Context("with missing properties in the JSON", func() {
		 	It("returns an error", func() {
		 		By("sending empty JSON")
		 		jsonBuffer = bytes.NewBufferString("{}")

		 		req, err := http.NewRequest("POST", "/v1/apps/someEnv/someOrg/someSpace/someApp", jsonBuffer)
		 		Expect(err).ToNot(HaveOccurred())

		 		req.SetBasicAuth(username, password)

		 		router.ServeHTTP(resp, req)

		 		Expect(resp.Code).To(Equal(500))
		 		Expect(resp.Body.String()).To(ContainSubstring("The following properties are missing: artifact_url"))
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

	 	Context("when custom manifest information is given in the request body", func() {
	 		It("properly decodes base64 encoding of the provided manifest information", func() {
	 			eventManager.EmitCall.Returns.Error = nil
	 			deployer.DeployCall.Write.Output = "push succeeded"
	 			deployer.DeployCall.Returns.Error = nil

	 			deploymentInfo.Manifest = "manifest-" + randomizer.StringRunes(10)

	 			By("base64 encoding a manifest")
	 			base64Manifest := base64.StdEncoding.EncodeToString([]byte(deploymentInfo.Manifest))

	 			By("including manifest in the JSON")
	 			jsonBuffer = bytes.NewBufferString(fmt.Sprintf(`{
	 					"artifact_url": "%s",
	 					"manifest": "%s"
	 				}`,
	 				artifactURL,
	 				base64Manifest,
	 			))

	 			req, err := http.NewRequest("POST", apiURL, jsonBuffer)
	 			Expect(err).ToNot(HaveOccurred())

	 			req.SetBasicAuth(username, password)

	 			router.ServeHTTP(resp, req)

	 			Expect(resp.Code).To(Equal(200))
	 			Expect(resp.Body.String()).To(ContainSubstring("push succeeded"))
	 			Expect(eventManager.EmitCall.TimesCalled).To(Equal(2))
	 			Expect(deployer.DeployCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
	 		})

	 		It("returns an error if the provided manifest information is not base64 encoded", func() {
	 			deploymentInfo.Manifest = "manifest-" + randomizer.StringRunes(10)

	 			By("not base64 encoding a manifest")

	 			By("including manifest in the JSON")
	 			jsonBuffer = bytes.NewBufferString(fmt.Sprintf(`{
	 					"artifact_url": "%s",
	 					"manifest": "%s"
	 				}`,
	 				artifactURL,
	 				deploymentInfo.Manifest,
	 			))

	 			req, err := http.NewRequest("POST", apiURL, jsonBuffer)
	 			Expect(err).ToNot(HaveOccurred())

	 			req.SetBasicAuth(username, password)

	 			router.ServeHTTP(resp, req)
	 			Expect(resp.Code).To(Equal(500))
	 			Expect(eventManager.EmitCall.TimesCalled).To(Equal(0))
	 		})
	 	})
	})

	Describe("deploy zip", func() {
		Context("when all applications start correctly", func() {
			It("accepts the request with a 200 OK", func() {

			})
		})

		Context("when fetcher fails", func() {
			It("rejects the request with a 500 Internal Server Error", func() {

			})
		})
		Context("when manifest file cannot be found in the extracted zip", func() {
			It("returns an error and status code 400", func() {

			})
		})
		Context("push fails", func() {
			It("rejects the request with a 500 Internal Server Error", func() {

			})
		})
		Context("deploy event handler fails", func() {
			It("rejects the request with a 500 Internal Server Error", func() {

			})
		})
	})

  Describe("Common Functionality", func() {
		Context("when authentication is required and a username and password are provided", func() {
			It("accepts the request with a 200 OK", func() {
				 			eventManager.EmitCall.Returns.Error = nil
				 			deployer.DeployCall.Write.Output = "push succeeded"
				 			deployer.DeployCall.Returns.Error = nil

				 			By("setting authenticate to true")
				 			controller.Config.Environments[environment] = config.Environment{Authenticate: true}

				 			req, err := http.NewRequest("POST", apiURL, jsonBuffer)
				 			Expect(err).ToNot(HaveOccurred())

				 			By("setting basic auth")
				 			req.SetBasicAuth(username, password)

				 			router.ServeHTTP(resp, req)

				 			Expect(resp.Code).To(Equal(200))
				 			Expect(resp.Body.String()).To(ContainSubstring("push succeeded"))
				 			Expect(eventManager.EmitCall.TimesCalled).To(Equal(2))
				 			Expect(deployer.DeployCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
			})

			Context("when authentication is required and a username and password is not provided", func() {
	    	It("rejects the request with a 401 unauthorized", func() {
					By("setting authenticate to true")
						controller.Config.Environments[environment] = config.Environment{Authenticate: true}

			 			req, err := http.NewRequest("POST", apiURL, jsonBuffer)
			 			Expect(err).ToNot(HaveOccurred())

			 			By("not setting basic auth")

			 			router.ServeHTTP(resp, req)
			 			Expect(resp.Code).To(Equal(401))
			 		})
			 	})

			Context("username and password are provided", func() {
			 		It("accepts the request with a 200 OK", func() {
			 			eventManager.EmitCall.Returns.Error = nil
			 			deployer.DeployCall.Write.Output = "push succeeded"
			 			deployer.DeployCall.Returns.Error = nil

			 			By("setting authenticate to false")
			 			controller.Config.Environments[environment] = config.Environment{Authenticate: false}

			 			req, err := http.NewRequest("POST", apiURL, jsonBuffer)
			 			Expect(err).ToNot(HaveOccurred())

			 			By("setting basic auth")
			 			req.SetBasicAuth(username, password)

			 			router.ServeHTTP(resp, req)

			 			Expect(resp.Code).To(Equal(200))
			 			Expect(resp.Body.String()).To(ContainSubstring("push succeeded"))
			 			Expect(eventManager.EmitCall.TimesCalled).To(Equal(2))
			 			Expect(deployer.DeployCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
			 	})
			})

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

			Context("deployer prechecker fails", func() {
	    	It("rejects the request with a 500 Internal Server Error", func() {
					prechecker.AssertAllFoundationsUpCall.Returns.Error = errors.New(deployAborted)

					err, statusCode := deployer.Deploy(req, environmentName, org, space, appName, buffer)
					Expect(err).To(MatchError("Deploy aborted, one or more CF foundations unavailable"))
					Expect(statusCode).To(Equal(500))

					Expect(prechecker.AssertAllFoundationsUpCall.Received.Environment).To(Equal(environments[environmentName]))
				})
			})
		})

		Describe("deployment output", func() {
		 It("shows the user deployment info properties", func() {
			 eventManager.EmitCall.Returns.Error = nil
			 deployer.DeployCall.Returns.Error = nil

			 req, err := http.NewRequest("POST", apiURL, jsonBuffer)
			 Expect(err).ToNot(HaveOccurred())

			 req.SetBasicAuth(username, password)

			 router.ServeHTTP(resp, req)
			 Expect(resp.Code).To(Equal(200))

			 result := resp.Body.String()
			 Expect(result).To(ContainSubstring(artifactURL))
			 Expect(result).To(ContainSubstring(username))
			 Expect(result).To(ContainSubstring(environment))
			 Expect(result).To(ContainSubstring(org))
			 Expect(result).To(ContainSubstring(space))
			 Expect(result).To(ContainSubstring(appName))

			 Expect(eventManager.EmitCall.TimesCalled).To(Equal(2))
			 Expect(deployer.DeployCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
		 })
		})
	})
})
