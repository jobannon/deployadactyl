package controller_test

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"io/ioutil"

	"os"

	"github.com/compozed/deployadactyl/config"
	. "github.com/compozed/deployadactyl/controller"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	//"github.com/compozed/deployadactyl/structs"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/op/go-logging"
)

var _ = Describe("Controller", func() {

	var (
		deployer        *mocks.Deployer
		silentDeployer  *mocks.Deployer
		eventManager    *mocks.EventManager
		errorFinder     *mocks.ErrorFinder
		stopController  *mocks.StopController
		startController *mocks.StartController
		pushController  *mocks.PushController
		controller      *Controller
		logBuffer       *Buffer

		appName     string
		environment string
		org         string
		space       string
		uuid        string
		byteBody    []byte
		server      *httptest.Server
	)

	BeforeEach(func() {
		logBuffer = NewBuffer()
		appName = "appName-" + randomizer.StringRunes(10)
		environment = "environment-" + randomizer.StringRunes(10)
		org = "org-" + randomizer.StringRunes(10)
		space = "non-prod"
		uuid = "uuid-" + randomizer.StringRunes(10)

		eventManager = &mocks.EventManager{}
		deployer = &mocks.Deployer{}
		silentDeployer = &mocks.Deployer{}
		pushController = &mocks.PushController{}
		stopController = &mocks.StopController{}
		startController = &mocks.StartController{}

		errorFinder = &mocks.ErrorFinder{}
		controller = &Controller{
			Deployer:        deployer,
			SilentDeployer:  silentDeployer,
			Log:             logger.DefaultLogger(logBuffer, logging.DEBUG, "api_test"),
			PushController:  pushController,
			StopController:  stopController,
			StartController: startController,
			EventManager:    eventManager,
			Config:          config.Config{},
			ErrorFinder:     errorFinder,
		}
	})

	Describe("RunDeploymentViaHttp handler", func() {
		var (
			router        *gin.Engine
			resp          *httptest.ResponseRecorder
			jsonBuffer    *bytes.Buffer
			foundationURL string
		)
		BeforeEach(func() {
			router = gin.New()
			resp = httptest.NewRecorder()
			jsonBuffer = &bytes.Buffer{}

			router.POST("/v2/deploy/:environment/:org/:space/:appName", controller.RunDeploymentViaHttp)

			server = httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
				byteBody, _ = ioutil.ReadAll(req.Body)
				req.Body.Close()
			}))

			silentDeployUrl := server.URL + "/v1/apps/" + os.Getenv("SILENT_DEPLOY_ENVIRONMENT")
			os.Setenv("SILENT_DEPLOY_URL", silentDeployUrl)
		})
		AfterEach(func() {
			server.Close()
		})

		Context("when deployer succeeds", func() {
			It("deploys and returns http.StatusOK", func() {
				foundationURL = fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, space, appName)

				req, err := http.NewRequest("POST", foundationURL, jsonBuffer)
				req.Header.Set("Content-Type", "application/zip")

				Expect(err).ToNot(HaveOccurred())

				pushController.RunDeploymentCall.Returns.DeployResponse = I.DeployResponse{
					StatusCode: http.StatusOK,
				}
				pushController.RunDeploymentCall.Writes = "deploy success"

				router.ServeHTTP(resp, req)

				Eventually(resp.Code).Should(Equal(http.StatusOK))
				Eventually(resp.Body).Should(ContainSubstring("deploy success"))

				Eventually(pushController.RunDeploymentCall.Received.Deployment.CFContext.Environment).Should(Equal(environment))
				Eventually(pushController.RunDeploymentCall.Received.Deployment.CFContext.Organization).Should(Equal(org))
				Eventually(pushController.RunDeploymentCall.Received.Deployment.CFContext.Space).Should(Equal(space))
				Eventually(pushController.RunDeploymentCall.Received.Deployment.CFContext.Application).Should(Equal(appName))
			})

			It("does not run silent deploy when environment other than non-prop", func() {
				foundationURL = fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, "not-non-prod", appName)

				req, err := http.NewRequest("POST", foundationURL, jsonBuffer)
				req.Header.Set("Content-Type", "application/zip")

				Expect(err).ToNot(HaveOccurred())

				pushController.RunDeploymentCall.Returns.DeployResponse = I.DeployResponse{
					StatusCode: http.StatusOK,
				}
				pushController.RunDeploymentCall.Writes = "deploy success"

				router.ServeHTTP(resp, req)

				Eventually(resp.Code).Should(Equal(http.StatusOK))
				Eventually(resp.Body).Should(ContainSubstring("deploy success"))

				Eventually(len(byteBody)).Should(Equal(0))
			})
		})

		Context("when deployer fails", func() {
			It("doesn't deploy and gives http.StatusInternalServerError", func() {
				foundationURL = fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, space, appName)

				req, err := http.NewRequest("POST", foundationURL, jsonBuffer)
				req.Header.Set("Content-Type", "application/zip")

				Expect(err).ToNot(HaveOccurred())

				pushController.RunDeploymentCall.Returns.DeployResponse = I.DeployResponse{
					Error:      errors.New("bork"),
					StatusCode: http.StatusInternalServerError,
				}

				router.ServeHTTP(resp, req)

				Eventually(resp.Code).Should(Equal(http.StatusInternalServerError))
				Eventually(resp.Body).Should(ContainSubstring("bork"))
			})
		})

		Context("when parameters are added to the url", func() {
			It("does not return an error", func() {
				foundationURL = fmt.Sprintf("/v2/deploy/%s/%s/%s/%s?broken=false", environment, org, space, appName)

				req, err := http.NewRequest("POST", foundationURL, jsonBuffer)
				req.Header.Set("Content-Type", "application/zip")

				Expect(err).ToNot(HaveOccurred())

				deployer.DeployCall.Write.Output = "deploy success"
				deployer.DeployCall.Returns.StatusCode = http.StatusOK

				router.ServeHTTP(resp, req)

				Eventually(resp.Code).Should(Equal(http.StatusOK))
				Expect(pushController.RunDeploymentCall.Received.Deployment).ToNot(BeNil())
			})
		})
	})

	Describe("PutRequestHandler", func() {
		var (
			router     *gin.Engine
			resp       *httptest.ResponseRecorder
			jsonBuffer *bytes.Buffer
		)

		BeforeEach(func() {
			router = gin.New()
			resp = httptest.NewRecorder()
			jsonBuffer = &bytes.Buffer{}

			router.PUT("/v2/deploy/:environment/:org/:space/:appName", controller.PutRequestHandler)

			server = httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
				byteBody, _ = ioutil.ReadAll(req.Body)
				req.Body.Close()
			}))
		})

		AfterEach(func() {
			server.Close()
		})

		Context("when state is set to stopped", func() {
			Context("when stop succeeds", func() {
				It("returns http status.OK", func() {
					foundationURL := fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, space, appName)
					jsonBuffer = bytes.NewBufferString(`{"state": "stopped"}`)

					req, err := http.NewRequest("PUT", foundationURL, jsonBuffer)
					req.Header.Set("Content-Type", "application/json")

					Expect(err).ToNot(HaveOccurred())

					router.ServeHTTP(resp, req)

					Eventually(resp.Code).Should(Equal(http.StatusOK))
				})
			})

			It("logs request origination address", func() {
				foundationURL := fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, space, appName)
				jsonBuffer = bytes.NewBufferString(`{"state": "stopped"}`)

				req, _ := http.NewRequest("PUT", foundationURL, jsonBuffer)
				req.Header.Set("Content-Type", "application/json")

				router.ServeHTTP(resp, req)

				Eventually(logBuffer).Should(Say("PUT Request originated from"))
			})

			It("calls StopDeployment with a Deployment", func() {
				foundationURL := fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, space, appName)
				jsonBuffer = bytes.NewBufferString(`{"state": "stopped"}`)

				req, err := http.NewRequest("PUT", foundationURL, jsonBuffer)
				req.Header.Set("Content-Type", "application/json")

				Expect(err).ToNot(HaveOccurred())

				router.ServeHTTP(resp, req)

				Expect(stopController.StopDeploymentCall.Received.Deployment).ToNot(BeNil())
			})

			It("calls StopDeployment with correct CFContext", func() {
				foundationURL := fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, space, appName)
				jsonBuffer = bytes.NewBufferString(`{"state": "stopped"}`)

				req, err := http.NewRequest("PUT", foundationURL, jsonBuffer)
				req.Header.Set("Content-Type", "application/json")

				Expect(err).ToNot(HaveOccurred())

				router.ServeHTTP(resp, req)

				cfContext := stopController.StopDeploymentCall.Received.Deployment.CFContext
				Expect(cfContext.Environment).To(Equal(environment))
				Expect(cfContext.Space).To(Equal(space))
				Expect(cfContext.Organization).To(Equal(org))
				Expect(cfContext.Application).To(Equal(appName))
			})

			It("calls StopDeployment with correct authorization", func() {
				foundationURL := fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, space, appName)
				jsonBuffer = bytes.NewBufferString(`{"state": "stopped"}`)

				req, err := http.NewRequest("PUT", foundationURL, jsonBuffer)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Basic bXlVc2VyOm15UGFzc3dvcmQ=")

				Expect(err).ToNot(HaveOccurred())

				router.ServeHTTP(resp, req)

				auth := stopController.StopDeploymentCall.Received.Deployment.Authorization
				Expect(auth.Username).To(Equal("myUser"))
				Expect(auth.Password).To(Equal("myPassword"))
			})

			It("writes the process output to the response", func() {
				foundationURL := fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, space, appName)
				jsonBuffer = bytes.NewBufferString(`{"state": "stopped"}`)

				req, err := http.NewRequest("PUT", foundationURL, jsonBuffer)
				req.Header.Set("Content-Type", "application/json")

				Expect(err).ToNot(HaveOccurred())

				stopController.StopDeploymentCall.Writes = "this is the process output"
				router.ServeHTTP(resp, req)

				bytes, _ := ioutil.ReadAll(resp.Body)
				Expect(string(bytes)).To(ContainSubstring("this is the process output"))
			})

			It("passes the data in the request body", func() {
				foundationURL := fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, space, appName)
				jsonBuffer = bytes.NewBufferString(`{"state": "stopped", "data": {"user_id": "jhodo", "group": "XP_IS_CHG" }}`)

				req, err := http.NewRequest("PUT", foundationURL, jsonBuffer)
				req.Header.Set("Content-Type", "application/json")

				Expect(err).ToNot(HaveOccurred())

				router.ServeHTTP(resp, req)

				Expect(stopController.StopDeploymentCall.Received.Data["user_id"]).To(Equal("jhodo"))
				Expect(stopController.StopDeploymentCall.Received.Data["group"]).To(Equal("XP_IS_CHG"))
			})

			Context("if requested state is not 'stop'", func() {
				It("does not call StopDeployment", func() {
					foundationURL := fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, space, appName)
					jsonBuffer = bytes.NewBufferString(`{"state": "started"}`)

					req, err := http.NewRequest("PUT", foundationURL, jsonBuffer)
					req.Header.Set("Content-Type", "application/json")

					Expect(err).ToNot(HaveOccurred())

					router.ServeHTTP(resp, req)

					Expect(stopController.StopDeploymentCall.Called).To(Equal(false))
				})
			})
		})

		Context("when state is set to started", func() {
			It("calls StartDeployment with a Deployment", func() {
				foundationURL := fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, space, appName)
				jsonBuffer = bytes.NewBufferString(`{"state": "started"}`)

				req, err := http.NewRequest("PUT", foundationURL, jsonBuffer)
				req.Header.Set("Content-Type", "application/json")

				Expect(err).ToNot(HaveOccurred())

				router.ServeHTTP(resp, req)

				Expect(startController.StartDeploymentCall.Received.Deployment).ToNot(BeNil())
			})

			It("calls StartDeployment with correct CFContext", func() {
				foundationURL := fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, space, appName)
				jsonBuffer = bytes.NewBufferString(`{"state": "started"}`)

				req, err := http.NewRequest("PUT", foundationURL, jsonBuffer)
				req.Header.Set("Content-Type", "application/json")

				Expect(err).ToNot(HaveOccurred())

				router.ServeHTTP(resp, req)

				cfContext := startController.StartDeploymentCall.Received.Deployment.CFContext
				Expect(cfContext.Environment).To(Equal(environment))
				Expect(cfContext.Space).To(Equal(space))
				Expect(cfContext.Organization).To(Equal(org))
				Expect(cfContext.Application).To(Equal(appName))
			})

			It("calls StartDeployment with correct authorization", func() {
				foundationURL := fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, space, appName)
				jsonBuffer = bytes.NewBufferString(`{"state": "started"}`)

				req, err := http.NewRequest("PUT", foundationURL, jsonBuffer)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Basic bXlVc2VyOm15UGFzc3dvcmQ=")

				Expect(err).ToNot(HaveOccurred())

				router.ServeHTTP(resp, req)

				auth := startController.StartDeploymentCall.Received.Deployment.Authorization
				Expect(auth.Username).To(Equal("myUser"))
				Expect(auth.Password).To(Equal("myPassword"))
			})

			Context("if requested state is not 'start'", func() {
				It("does not call StartDeployment", func() {
					foundationURL := fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, space, appName)
					jsonBuffer = bytes.NewBufferString(`{"state": "stopped"}`)

					req, err := http.NewRequest("PUT", foundationURL, jsonBuffer)
					req.Header.Set("Content-Type", "application/json")

					Expect(err).ToNot(HaveOccurred())

					router.ServeHTTP(resp, req)

					Expect(startController.StartDeploymentCall.Called).To(Equal(false))
				})
			})
		})
	})

})
