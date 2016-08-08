package controller_test

import (
	"bytes"
	"fmt"
	"net/http/httptest"

	"github.com/compozed/deployadactyl/config"
	. "github.com/compozed/deployadactyl/controller"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	"github.com/op/go-logging"
)

var _ = Describe("Controller", func() {

	var (
		controller   *Controller
		deployer     *mocks.Deployer
		eventManager *mocks.EventManager
		fetcher      *mocks.Fetcher
		router       *gin.Engine
		resp         *httptest.ResponseRecorder

		environment     string
		org             string
		space           string
		appName         string
		defaultUsername string
		defaultPassword string
		apiURL          string

		jsonBuffer *bytes.Buffer
	)

	BeforeEach(func() {
		deployer = &mocks.Deployer{}
		eventManager = &mocks.EventManager{}
		fetcher = &mocks.Fetcher{}

		jsonBuffer = &bytes.Buffer{}

		envMap := map[string]config.Environment{}
		envMap["Test"] = config.Environment{Foundations: []string{"api1.example.com", "api2.example.com"}}
		envMap["Prod"] = config.Environment{Foundations: []string{"api3.example.com", "api4.example.com"}}

		environment = "environment-" + randomizer.StringRunes(10)
		org = "org-" + randomizer.StringRunes(10)
		space = "space-" + randomizer.StringRunes(10)
		appName = "appName-" + randomizer.StringRunes(10)
		defaultUsername = "defaultUsername-" + randomizer.StringRunes(10)
		defaultPassword = "defaultPassword-" + randomizer.StringRunes(10)
		jsonBuffer = bytes.NewBufferString("jsonBuffer-" + randomizer.StringRunes(10))

		c := config.Config{
			Username:     defaultUsername,
			Password:     defaultPassword,
			Environments: envMap,
		}

		controller = &Controller{
			Config:       c,
			Deployer:     deployer,
			Log:          logger.DefaultLogger(GinkgoWriter, logging.DEBUG, "api_test"),
			EventManager: eventManager,
			Fetcher:      fetcher,
		}

		apiURL = fmt.Sprintf("/v1/apps/%s/%s/%s/%s",
			environment,
			org,
			space,
			appName,
		)

		router = gin.New()
		resp = httptest.NewRecorder()

		router.POST("/v1/apps/:environment/:org/:space/:appName", controller.Deploy)

	})

	Context("the controller receives a request that has a mime type application/json", func() {
		Describe("successful deployments without missing properties with a remote artifact url", func() {
			It("deploys successfully with no error message and a status code of 200 OK", func() {

			})
		})

		Describe("failed deployments", func() {
			It("fails to build the application", func() {

				By("returning an error message and a status code that is not 2xx")
			})
		})
	})

	Context("the controller receives a request that has a mime type application/zip", func() {
		Describe("successful deployments with a local zip file", func() {
			It("deploys successfully with no error message and a status code of 200 OK", func() {

			})
		})

		Describe("failed deployments", func() {
			It("cannot form the zip file", func() {

			})

			It("cannot process the zip file", func() {

			})

			It("fails to build", func() {

				By("returning an error message and a status code that is not 2xx")
			})
		})
	})

	Context("the controller receives a request that is not a recognised mime type", func() {
		It("does not deploy", func() {

		})
	})

	// var (
	// 	controller     *Controller
	// 	deployer       *mocks.Deployer
	// 	router         *gin.Engine
	// 	resp           *httptest.ResponseRecorder
	// 	eventManager   *mocks.EventManager
	// 	fetcher        *mocks.Fetcher

	// 	jsonBuffer *bytes.Buffer

	// 	deploymentInfo S.DeploymentInfo

	// 	artifactURL     string
	// 	environment     string
	// 	org             string
	// 	space           string
	// 	appName         string
	// 	username        string
	// 	password        string
	// 	defaultUsername string
	// 	defaultPassword string
	// 	apiURL          string
	// 	uuid            string
	// 	skipSSL         bool
	// )

	// BeforeEach(func() {
	// 	deployer = &mocks.Deployer{}
	// 	eventManager = &mocks.EventManager{}
	// 	fetcher = &mocks.Fetcher{}

	// 	jsonBuffer = &bytes.Buffer{}

	// 	envMap := map[string]config.Environment{}
	// 	envMap["Test"] = config.Environment{Foundations: []string{"api1.example.com", "api2.example.com"}}
	// 	envMap["Prod"] = config.Environment{Foundations: []string{"api3.example.com", "api4.example.com"}}

	// 	artifactURL = "artifactURL-" + randomizer.StringRunes(10)
	// 	environment = "environment-" + randomizer.StringRunes(10)
	// 	org = "org-" + randomizer.StringRunes(10)
	// 	space = "space-" + randomizer.StringRunes(10)
	// 	appName = "appName-" + randomizer.StringRunes(10)
	// 	username = "username-" + randomizer.StringRunes(10)
	// 	password = "password-" + randomizer.StringRunes(10)
	// 	defaultUsername = "defaultUsername-" + randomizer.StringRunes(10)
	// 	defaultPassword = "defaultPassword-" + randomizer.StringRunes(10)
	// 	uuid = "uuid-" + randomizer.StringRunes(123)

	// 	c := config.Config{
	// 		Username:     defaultUsername,
	// 		Password:     defaultPassword,
	// 		Environments: envMap,
	// 	}

	// 	controller = &Controller{
	// 		Deployer:     deployer,
	// 		Log:          logger.DefaultLogger(GinkgoWriter, logging.DEBUG, "api_test"),
	// 		Config:       c,
	// 		EventManager: eventManager,
	// 		Fetcher:      fetcher,
	// 	}

	// 	apiURL = fmt.Sprintf("/v1/apps/%s/%s/%s/%s",
	// 		environment,
	// 		org,
	// 		space,
	// 		appName,
	// 	)

	// 	deploymentInfo = S.DeploymentInfo{
	// 		ArtifactURL: artifactURL,
	// 		Username:    username,
	// 		Password:    password,
	// 		Environment: environment,
	// 		Org:         org,
	// 		Space:       space,
	// 		AppName:     appName,
	// 		UUID:        uuid,
	// 		SkipSSL:     skipSSL,
	// 	}

	// 	//randomizerMock.RandomizeCall.Returns.Runes = uuid

	// 	router = gin.New()
	// 	resp = httptest.NewRecorder()

	// 	router.POST("/v1/apps/:environment/:org/:space/:appName", controller.Deploy)

	// 	jsonBuffer = bytes.NewBufferString(fmt.Sprintf(`{
	// 			"artifact_url": "%s"
	// 		}`,
	// 		artifactURL,
	// 	))
	// })

	// Describe("missing properties in the JSON", func() {
	// 	It("returns an error", func() {
	// 		By("sending empty JSON")
	// 		jsonBuffer = bytes.NewBufferString("{}")

	// 		req, err := http.NewRequest("POST", "/v1/apps/someEnv/someOrg/someSpace/someApp", jsonBuffer)
	// 		Expect(err).ToNot(HaveOccurred())

	// 		req.SetBasicAuth(username, password)

	// 		router.ServeHTTP(resp, req)

	// 		Expect(resp.Code).To(Equal(500))
	// 		Expect(resp.Body.String()).To(ContainSubstring("The following properties are missing: artifact_url"))
	// 	})
	// })

	// Describe("authentication", func() {
	// 	Context("username and password are provided", func() {
	// 		It("accepts the request with a 200 OK", func() {
	// 			eventManager.EmitCall.Returns.Error = nil
	// 			deployer.DeployCall.Write.Output = "push succeeded"
	// 			deployer.DeployCall.Returns.Error = nil

	// 			By("setting authenticate to true")
	// 			controller.Config.Environments[environment] = config.Environment{Authenticate: true}

	// 			req, err := http.NewRequest("POST", apiURL, jsonBuffer)
	// 			Expect(err).ToNot(HaveOccurred())

	// 			By("setting basic auth")
	// 			req.SetBasicAuth(username, password)

	// 			router.ServeHTTP(resp, req)

	// 			Expect(resp.Code).To(Equal(200))
	// 			Expect(resp.Body.String()).To(ContainSubstring("push succeeded"))
	// 			Expect(eventManager.EmitCall.TimesCalled).To(Equal(2))
	// 			Expect(deployer.DeployCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
	// 		})
	// 	})

	// 	Context("when username and password are not provided", func() {
	// 		It("rejects the request with a 401 unauthorized", func() {
	// 			By("setting authenticate to true")
	// 			controller.Config.Environments[environment] = config.Environment{Authenticate: true}

	// 			req, err := http.NewRequest("POST", apiURL, jsonBuffer)
	// 			Expect(err).ToNot(HaveOccurred())

	// 			By("not setting basic auth")

	// 			router.ServeHTTP(resp, req)
	// 			Expect(resp.Code).To(Equal(401))
	// 		})
	// 	})

	// 	Context("username and password are provided", func() {
	// 		It("accepts the request with a 200 OK", func() {
	// 			eventManager.EmitCall.Returns.Error = nil
	// 			deployer.DeployCall.Write.Output = "push succeeded"
	// 			deployer.DeployCall.Returns.Error = nil

	// 			By("setting authenticate to false")
	// 			controller.Config.Environments[environment] = config.Environment{Authenticate: false}

	// 			req, err := http.NewRequest("POST", apiURL, jsonBuffer)
	// 			Expect(err).ToNot(HaveOccurred())

	// 			By("setting basic auth")
	// 			req.SetBasicAuth(username, password)

	// 			router.ServeHTTP(resp, req)

	// 			Expect(resp.Code).To(Equal(200))
	// 			Expect(resp.Body.String()).To(ContainSubstring("push succeeded"))
	// 			Expect(eventManager.EmitCall.TimesCalled).To(Equal(2))
	// 			Expect(deployer.DeployCall.Received.DeploymentInfo).To(Equal(deploymentInfo))

	// 		})

	// 		Context("username and password are not provided", func() {
	// 			It("accepts the request with a 200 OK", func() {
	// 				eventManager.EmitCall.Returns.Error = nil
	// 				deployer.DeployCall.Write.Output = "push succeeded"
	// 				deployer.DeployCall.Returns.Error = nil

	// 				By("setting authenticate to false")
	// 				controller.Config.Environments[environment] = config.Environment{Authenticate: false}

	// 				req, err := http.NewRequest("POST", apiURL, jsonBuffer)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				By("not setting basic auth and setting the default username and password")
	// 				deploymentInfo.Username = defaultUsername
	// 				deploymentInfo.Password = defaultPassword

	// 				router.ServeHTTP(resp, req)

	// 				Expect(resp.Code).To(Equal(200))
	// 				Expect(eventManager.EmitCall.TimesCalled).To(Equal(2))
	// 				Expect(deployer.DeployCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
	// 			})
	// 		})
	// 	})
	// })

	// Describe("successful deployments without missing properties with a remote artifact url", func() {
	// 	It("returns a 200 OK and responds with the output of the push command", func() {
	// 		eventManager.EmitCall.Returns.Error = nil
	// 		deployer.DeployCall.Write.Output = "push succeeded"
	// 		deployer.DeployCall.Returns.Error = nil

	// 		req, err := http.NewRequest("POST", apiURL, jsonBuffer)
	// 		Expect(err).ToNot(HaveOccurred())

	// 		req.SetBasicAuth(username, password)

	// 		router.ServeHTTP(resp, req)

	// 		Expect(resp.Code).To(Equal(200))
	// 		Expect(resp.Body.String()).To(ContainSubstring("push succeeded"))
	// 		Expect(eventManager.EmitCall.TimesCalled).To(Equal(2))
	// 		Expect(deployer.DeployCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
	// 	})

	// 	Context("when custom manifest information is given in the request body", func() {
	// 		It("properly decodes base64 encoding of the provided manifest information", func() {
	// 			eventManager.EmitCall.Returns.Error = nil
	// 			deployer.DeployCall.Write.Output = "push succeeded"
	// 			deployer.DeployCall.Returns.Error = nil

	// 			deploymentInfo.Manifest = "manifest-" + randomizer.StringRunes(10)

	// 			By("base64 encoding a manifest")
	// 			base64Manifest := base64.StdEncoding.EncodeToString([]byte(deploymentInfo.Manifest))

	// 			By("including manifest in the JSON")
	// 			jsonBuffer = bytes.NewBufferString(fmt.Sprintf(`{
	// 					"artifact_url": "%s",
	// 					"manifest": "%s"
	// 				}`,
	// 				artifactURL,
	// 				base64Manifest,
	// 			))

	// 			req, err := http.NewRequest("POST", apiURL, jsonBuffer)
	// 			Expect(err).ToNot(HaveOccurred())

	// 			req.SetBasicAuth(username, password)

	// 			router.ServeHTTP(resp, req)

	// 			Expect(resp.Code).To(Equal(200))
	// 			Expect(resp.Body.String()).To(ContainSubstring("push succeeded"))
	// 			Expect(eventManager.EmitCall.TimesCalled).To(Equal(2))
	// 			Expect(deployer.DeployCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
	// 		})

	// 		It("returns an error if the provided manifest information is not base64 encoded", func() {
	// 			deploymentInfo.Manifest = "manifest-" + randomizer.StringRunes(10)

	// 			By("not base64 encoding a manifest")

	// 			By("including manifest in the JSON")
	// 			jsonBuffer = bytes.NewBufferString(fmt.Sprintf(`{
	// 					"artifact_url": "%s",
	// 					"manifest": "%s"
	// 				}`,
	// 				artifactURL,
	// 				deploymentInfo.Manifest,
	// 			))

	// 			req, err := http.NewRequest("POST", apiURL, jsonBuffer)
	// 			Expect(err).ToNot(HaveOccurred())

	// 			req.SetBasicAuth(username, password)

	// 			router.ServeHTTP(resp, req)
	// 			Expect(resp.Code).To(Equal(500))
	// 			Expect(eventManager.EmitCall.TimesCalled).To(Equal(0))
	// 		})
	// 	})
	// })

	// Describe("successful deployments of a local application", func() {
	// 	It("returns a 200 OK and responds with the output of the push command", func() {
	// 	})

	// 	Context("when custom manifest information is given in a manifest file", func() {
	// 		It("properly decodes the provided manifest information", func() {
	// 		})

	// 		It("returns an error if the provided manifest information is invalid", func() {
	// 		})
	// 	})
	// })

	// Describe("when deployer fails", func() {
	// 	It("returns a 500 internal server error and responds with the output of the push command", func() {
	// 		eventManager.EmitCall.Returns.Error = nil

	// 		By("making deployer return an error")
	// 		deployer.DeployCall.Write.Output = "some awesome CF error\n"
	// 		deployer.DeployCall.Returns.Error = errors.New("bork")

	// 		req, err := http.NewRequest("POST", apiURL, jsonBuffer)
	// 		Expect(err).ToNot(HaveOccurred())

	// 		req.SetBasicAuth(username, password)

	// 		router.ServeHTTP(resp, req)

	// 		Expect(resp.Code).To(Equal(500))
	// 		Expect(resp.Body.String()).To(ContainSubstring("some awesome CF error\n"))
	// 		Expect(eventManager.EmitCall.TimesCalled).To(Equal(2))
	// 		Expect(deployer.DeployCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
	// 	})
	// })

	// Describe("deployment output", func() {
	// 	It("shows the user deployment info properties", func() {
	// 		eventManager.EmitCall.Returns.Error = nil
	// 		deployer.DeployCall.Returns.Error = nil

	// 		req, err := http.NewRequest("POST", apiURL, jsonBuffer)
	// 		Expect(err).ToNot(HaveOccurred())

	// 		req.SetBasicAuth(username, password)

	// 		router.ServeHTTP(resp, req)
	// 		Expect(resp.Code).To(Equal(200))

	// 		result := resp.Body.String()
	// 		Expect(result).To(ContainSubstring(artifactURL))
	// 		Expect(result).To(ContainSubstring(username))
	// 		Expect(result).To(ContainSubstring(environment))
	// 		Expect(result).To(ContainSubstring(org))
	// 		Expect(result).To(ContainSubstring(space))
	// 		Expect(result).To(ContainSubstring(appName))

	// 		Expect(eventManager.EmitCall.TimesCalled).To(Equal(2))
	// 		Expect(deployer.DeployCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
	// 	})
	// })
})
