package controller_test

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/compozed/deployadactyl/config"
	. "github.com/compozed/deployadactyl/controller"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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

		deployer.DeployCall.Received.EnvironmentName = environment
		deployer.DeployCall.Received.Org = org
		deployer.DeployCall.Received.Space = space
		deployer.DeployCall.Received.AppName = appName
		deployer.DeployCall.Received.Out = jsonBuffer

		deployer.DeployZipCall.Received.EnvironmentName = environment
		deployer.DeployZipCall.Received.Org = org
		deployer.DeployZipCall.Received.Space = space
		deployer.DeployZipCall.Received.AppName = appName
		deployer.DeployZipCall.Received.Out = jsonBuffer
	})

	Describe("the controller receives a request that has a mime type application/json", func() {
		Context("successful deployments without missing properties with a remote artifact url", func() {
			It("deploys successfully with no error message and a status code of 200 OK", func() {
				req, err := http.NewRequest("POST", apiURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")

				deployer.DeployCall.Received.Request = req
				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = 200

				router.ServeHTTP(resp, req)

				Expect(deployer.DeployCall.TimesCalled).To(Equal(1))
				Expect(resp.Code).To(Equal(200))
				Expect(resp.Body).To(ContainSubstring("deploy successful"))
			})
		})

		Context("failed deployments", func() {
			It("fails to build the application", func() {
				req, err := http.NewRequest("POST", apiURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")

				deployer.DeployCall.Received.Request = req

				By("returning an error message and a status code that is 500")
				deployer.DeployCall.Returns.Error = errors.New("internal server error")
				deployer.DeployCall.Returns.StatusCode = 500

				router.ServeHTTP(resp, req)

				Expect(deployer.DeployCall.TimesCalled).To(Equal(1))
				Expect(resp.Code).To(Equal(500))
				Expect(resp.Body).To(ContainSubstring("internal server error"))
			})
		})
	})

	Describe("the controller receives a request that has a mime type application/zip", func() {
		Context("successful deployments with a local zip file", func() {
			It("deploys successfully with no error message and a status code of 200 OK", func() {
				req, err := http.NewRequest("POST", apiURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Content-Type", "application/zip")

				deployer.DeployZipCall.Received.Request = req

				fetcher.FetchFromZipCall.Received.RequestBody = nil
				fetcher.FetchFromZipCall.Returns.AppPath = "appPath-" + randomizer.StringRunes(10)
				fetcher.FetchFromZipCall.Returns.Error = nil

				router.ServeHTTP(resp, req)

				Expect(deployer.DeployZipCall.TimesCalled).To(Equal(1))
				Expect(resp.Code).To(Equal(200))
				Expect(resp.Body).To(ContainSubstring("deploy successful"))
			})
		})

		Context("failed deployments", func() {
			It("request is empty", func() {
				req, err := http.NewRequest("POST", apiURL, nil)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Content-Type", "application/zip")

				deployer.DeployZipCall.Received.Request = req

				router.ServeHTTP(resp, req)

				Expect(deployer.DeployZipCall.TimesCalled).To(Equal(0))
				Expect(resp.Code).To(Equal(400))
				Expect(resp.Body).To(ContainSubstring("request body is empty"))
			})

			It("cannot process the zip file", func() {
				req, err := http.NewRequest("POST", apiURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Content-Type", "application/zip")

				deployer.DeployZipCall.Received.Request = req

				fetcher.FetchFromZipCall.Received.RequestBody = jsonBuffer.Bytes()
				fetcher.FetchFromZipCall.Returns.AppPath = ""
				fetcher.FetchFromZipCall.Returns.Error = errors.New("could not process zip file")

				router.ServeHTTP(resp, req)

				Expect(deployer.DeployZipCall.TimesCalled).To(Equal(0))
				Expect(resp.Code).To(Equal(500))
				Expect(resp.Body).To(ContainSubstring("could not process zip file"))
			})

			It("fails to build", func() {
				req, err := http.NewRequest("POST", apiURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Content-Type", "application/zip")

				deployer.DeployZipCall.Received.Request = req

				fetcher.FetchFromZipCall.Received.RequestBody = jsonBuffer.Bytes()
				fetcher.FetchFromZipCall.Returns.AppPath = "appPath-" + randomizer.StringRunes(10)
				fetcher.FetchFromZipCall.Returns.Error = nil

				By("returning an error message and a status code that is 500")
				deployer.DeployZipCall.Returns.Error = errors.New("internal server error")
				deployer.DeployZipCall.Returns.StatusCode = 500

				router.ServeHTTP(resp, req)

				Expect(deployer.DeployZipCall.TimesCalled).To(Equal(1))
				Expect(resp.Code).To(Equal(500))
				Expect(resp.Body).To(ContainSubstring("cannot deploy application"))
			})
		})
	})

	Describe("the controller receives a request that is not a recognized mime type", func() {
		It("does not deploy", func() {
			req, err := http.NewRequest("POST", apiURL, jsonBuffer)
			Expect(err).ToNot(HaveOccurred())
			req.Header.Set("Content-Type", "invalidContentType")

			deployer.DeployZipCall.Returns.Error = errors.New("internal server error")
			deployer.DeployZipCall.Returns.StatusCode = 400

			router.ServeHTTP(resp, req)

			Expect(deployer.DeployCall.TimesCalled).To(Equal(0))
			Expect(deployer.DeployZipCall.TimesCalled).To(Equal(0))
			Expect(resp.Code).To(Equal(400))
			Expect(resp.Body).To(ContainSubstring("content type 'invalidContentType' not supported"))
		})
	})
})
