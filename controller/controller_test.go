package controller_test

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"io/ioutil"

	"os"

	. "github.com/compozed/deployadactyl/controller"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/op/go-logging"
)

const (
	deployerNotEnoughCalls = "deployer didn't have the right number of calls"
)

var _ = Describe("Controller", func() {

	var (
		deployer   *mocks.Deployer
		controller *Controller
		router     *gin.Engine
		resp       *httptest.ResponseRecorder
		jsonBuffer *bytes.Buffer

		foundationURL string
		appName       string
		environment   string
		org           string
		space         string
		contentType   string
		byteBody      []byte
		server        *httptest.Server
	)

	BeforeEach(func() {
		deployer = &mocks.Deployer{}

		controller = &Controller{
			Deployer: deployer,
			Log:      logger.DefaultLogger(GinkgoWriter, logging.DEBUG, "api_test"),
		}

		router = gin.New()
		resp = httptest.NewRecorder()
		jsonBuffer = &bytes.Buffer{}

		appName = "appName-" + randomizer.StringRunes(10)
		environment = "environment-" + randomizer.StringRunes(10)
		org = "org-" + randomizer.StringRunes(10)
		space = "non-prod"
		contentType = "application/json"

		router.POST("/v1/deploy/:environment/:org/:space/:appName", controller.Deploy)

		server = httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			byteBody, _ = ioutil.ReadAll(req.Body)
		}))

		silentDeployUrl := server.URL + "/v1/apps/" + os.Getenv("SILENT_DEPLOY_SPACE") + "/%s/dev/%s"
		os.Setenv("SILENT_DEPLOY_URL", silentDeployUrl)
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("Deploy handler", func() {
		Context("when deployer succeeds", func() {
			It("deploys and returns http.StatusOK", func() {
				foundationURL = fmt.Sprintf("/v1/deploy/%s/%s/%s/%s", environment, org, space, appName)

				req, err := http.NewRequest("POST", foundationURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())

				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "deploy success"

				router.ServeHTTP(resp, req)

				Eventually(resp.Code).Should(Equal(http.StatusOK))
				Eventually(resp.Body).Should(ContainSubstring("deploy success"))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))
			})

			It("does not run silent deploy when environment other than non-prop", func() {
				foundationURL = fmt.Sprintf("/v1/deploy/%s/%s/%s/%s", environment, org, "not-non-prod", appName)

				req, err := http.NewRequest("POST", foundationURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())

				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "deploy success"

				router.ServeHTTP(resp, req)

				Eventually(resp.Code).Should(Equal(http.StatusOK))
				Eventually(resp.Body).Should(ContainSubstring("deploy success"))

				Eventually(len(byteBody)).Should(Equal(0))
			})
		})

		Context("when deployer fails", func() {
			It("doesn't deploy and gives http.StatusInternalServerError", func() {
				foundationURL = fmt.Sprintf("/v1/deploy/%s/%s/%s/%s", environment, org, space, appName)

				req, err := http.NewRequest("POST", foundationURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())

				deployer.DeployCall.Returns.Error = errors.New("bork")
				deployer.DeployCall.Returns.StatusCode = http.StatusInternalServerError

				router.ServeHTTP(resp, req)

				Eventually(resp.Code).Should(Equal(http.StatusInternalServerError))
				Eventually(resp.Body).Should(ContainSubstring("bork"))
			})
		})

		Context("when parameters are added to the url", func() {
			It("does not return an error", func() {
				foundationURL = fmt.Sprintf("/v1/deploy/%s/%s/%s/%s?broken=false", environment, org, space, appName)

				req, err := http.NewRequest("POST", foundationURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())

				deployer.DeployCall.Write.Output = "deploy success"
				deployer.DeployCall.Returns.StatusCode = http.StatusOK

				router.ServeHTTP(resp, req)

				Eventually(resp.Code).Should(Equal(http.StatusOK))
				Eventually(resp.Body).Should(ContainSubstring("deploy success"))
			})
		})

		Context("when NotSilentDeploy is called", func() {
			It("channel resolves true when no errors occur", func() {
				req := &http.Request{}
				reqChannel := make(chan DeployResponse)
				response := &bytes.Buffer{}

				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "deploy success"

				go controller.NotSilentDeploy(req, environment, org, space, appName, contentType, reqChannel, response)
				someVariable := <-reqChannel

				Eventually(someVariable.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))

			})

			It("channel resolves false when errors occur", func() {
				req := &http.Request{}
				reqChannel := make(chan DeployResponse)
				response := &bytes.Buffer{}

				deployer.DeployCall.Returns.Error = errors.New("bork")
				deployer.DeployCall.Returns.StatusCode = http.StatusInternalServerError
				deployer.DeployCall.Write.Output = "deploy failed"

				go controller.NotSilentDeploy(req, environment, org, space, appName, contentType, reqChannel, response)
				someVariable := <-reqChannel

				Eventually(someVariable.StatusCode).Should(Equal(http.StatusInternalServerError))
				Eventually(someVariable.Error.Error()).Should(Equal("bork"))
			})
		})

		Context("when SilentDeploy is called", func() {
			It("channel resolves true when no errors occur", func() {
				req := &http.Request{}
				reqChannel := make(chan DeployResponse)

				jsonBuffer = bytes.NewBufferString(`{
					"artifact_url": "https://artifactory.allstate.com/artifactory/libs-release-local/com/allstate/conveyor-test/little-timmy-env.zip",
					"data": {
					"user_id": "sys-cfdplyr",
					"group": "ServiceNow_DEV"
					}
				}`)

				req, _ = http.NewRequest("POST", fmt.Sprintf("/v1/apps/non-prod/%s/dev/%s", org, appName), jsonBuffer)

				go controller.SilentDeploy(req, org, appName, reqChannel)
				someVariable := <-reqChannel

				Eventually(someVariable.StatusCode).Should(Equal(http.StatusOK))
				Eventually(string(byteBody)).Should(ContainSubstring("little-timmy-env.zip"))

			})

			It("channel resolves false when errors occur", func() {
				req := &http.Request{}
				reqChannel := make(chan DeployResponse)

				server = httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
					res.WriteHeader(500)
				}))

				silentDeployUrl := server.URL + "/v1/apps/" + os.Getenv("SILENT_DEPLOY_SPACE") + "/%s/dev/%s"
				os.Setenv("SILENT_DEPLOY_URL", silentDeployUrl)

				jsonBuffer = bytes.NewBufferString(`{
					"artifact_url": "https://artifactory.allstate.com/artifactory/libs-release-local/com/allstate/conveyor-test/little-timmy-env.zip",
					"data": {
					"user_id": "sys-cfdplyr",
					"group": "ServiceNow_DEV"
					}
				}`)

				req, _ = http.NewRequest("POST", fmt.Sprintf("/v1/apps/non-prod/%s/dev/%s", org, appName), jsonBuffer)

				go controller.SilentDeploy(req, org, appName, reqChannel)
				someVariable := <-reqChannel

				Eventually(someVariable.StatusCode).Should(Equal(http.StatusInternalServerError))

			})
		})
	})
})
