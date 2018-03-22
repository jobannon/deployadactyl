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
	I "github.com/compozed/deployadactyl/interfaces"
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
		deployer             *mocks.Deployer
		silentDeployer       *mocks.Deployer
		pusherCreator        *mocks.PusherCreator
		pusherCreatorFactory *mocks.PusherCreatorFactory
		controller           *Controller
		router               *gin.Engine
		resp                 *httptest.ResponseRecorder
		jsonBuffer           *bytes.Buffer

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
		silentDeployer = &mocks.Deployer{}
		pusherCreatorFactory = &mocks.PusherCreatorFactory{}
		controller = &Controller{
			Deployer:             deployer,
			SilentDeployer:       silentDeployer,
			Log:                  logger.DefaultLogger(GinkgoWriter, logging.DEBUG, "api_test"),
			PusherCreatorFactory: pusherCreatorFactory,
		}
		pusherCreatorFactory.PusherCreatorCall.Returns.ActionCreator = pusherCreator

		router = gin.New()
		resp = httptest.NewRecorder()
		jsonBuffer = &bytes.Buffer{}

		appName = "appName-" + randomizer.StringRunes(10)
		environment = "environment-" + randomizer.StringRunes(10)
		org = "org-" + randomizer.StringRunes(10)
		space = "non-prod"
		contentType = "application/json"

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

	Describe("RunDeploymentViaHttp handler", func() {
		Context("when deployer succeeds", func() {
			It("deploys and returns http.StatusOK", func() {
				foundationURL = fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, space, appName)

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
				foundationURL = fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, "not-non-prod", appName)

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
				foundationURL = fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, space, appName)

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
				foundationURL = fmt.Sprintf("/v2/deploy/%s/%s/%s/%s?broken=false", environment, org, space, appName)

				req, err := http.NewRequest("POST", foundationURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())

				deployer.DeployCall.Write.Output = "deploy success"
				deployer.DeployCall.Returns.StatusCode = http.StatusOK

				router.ServeHTTP(resp, req)

				Eventually(resp.Code).Should(Equal(http.StatusOK))
				Eventually(resp.Body).Should(ContainSubstring("deploy success"))
			})
		})
	})

	Describe("RunDeployment", func() {
		Context("when verbose deployer is called", func() {
			It("deployer is provided correct authorization", func() {

				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				response := &bytes.Buffer{}

				deployment := &I.Deployment{
					Body: &[]byte{},
					Authorization: I.Authorization{
						Username: "username",
						Password: "password",
					},
					CFContext: I.CFContext{
						Environment:  environment,
						Organization: org,
						Space:        space,
						Application:  appName,
					},
				}
				deployResponse := controller.RunDeployment(deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.Authorization).Should(Equal(deployment.Authorization))
				//Eventually(deployer.DeployCall.Received.ContentType).Should(Equal(I.DeploymentType{JSON: true}))
				//Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				//Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				//Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))
				//
				//ret, _ := ioutil.ReadAll(response)
				//Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})

			It("deployer is provided the body", func() {

				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				response := &bytes.Buffer{}

				bodyBytes := []byte("a test body string")

				deployment := &I.Deployment{
					Body: &bodyBytes,
					Authorization: I.Authorization{
						Username: "username",
						Password: "password",
					},
					CFContext: I.CFContext{
						Environment:  environment,
						Organization: org,
						Space:        space,
						Application:  appName,
					},
				}
				deployResponse := controller.RunDeployment(deployment, response)
				receivedBody, _ := ioutil.ReadAll(deployer.DeployCall.Received.Body)
				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(receivedBody).Should(Equal(*deployment.Body))
				//Eventually(deployer.DeployCall.Received.ContentType).Should(Equal(I.DeploymentType{JSON: true}))
				//Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				//Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				//Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))
				//
				//ret, _ := ioutil.ReadAll(response)
				//Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})

			It("channel resolves when no errors occur", func() {

				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				response := &bytes.Buffer{}

				deployment := &I.Deployment{
					Body: &[]byte{},
					Type: I.DeploymentType{JSON: true},
					CFContext: I.CFContext{
						Environment:  environment,
						Organization: org,
						Space:        space,
						Application:  appName,
					},
				}
				deployResponse := controller.RunDeployment(deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.ContentType).Should(Equal(I.DeploymentType{JSON: true}))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})

			It("channel resolves when errors occur", func() {

				deployer.DeployCall.Returns.Error = errors.New("bork")
				deployer.DeployCall.Returns.StatusCode = http.StatusInternalServerError
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				response := &bytes.Buffer{}

				deployment := &I.Deployment{
					Body: &[]byte{},
					Type: I.DeploymentType{JSON: true},
					CFContext: I.CFContext{
						Environment:  environment,
						Organization: org,
						Space:        space,
						Application:  appName,
					},
				}
				deployResponse := controller.RunDeployment(deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusInternalServerError))
				Eventually(deployResponse.Error.Error()).Should(Equal("bork"))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.ContentType).Should(Equal(I.DeploymentType{JSON: true}))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})

			It("does not set the basic auth header if no credentials are passed", func() {
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				response := &bytes.Buffer{}

				deployment := &I.Deployment{
					Body: &[]byte{},
					Type: I.DeploymentType{JSON: true},
					CFContext: I.CFContext{
						Environment:  environment,
						Organization: org,
						Space:        space,
						Application:  appName,
					},
					Authorization: I.Authorization{
						Username: "",
						Password: "",
					},
				}
				controller.RunDeployment(deployment, response)

				Eventually(deployer.DeployCall.Received.Authorization.Username).Should(Equal(""))
				Eventually(deployer.DeployCall.Received.Authorization.Password).Should(Equal(""))

			})

			It("sets the basic auth header if credentials are passed", func() {
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				response := &bytes.Buffer{}

				deployment := &I.Deployment{
					Body: &[]byte{},
					Type: I.DeploymentType{JSON: true},
					CFContext: I.CFContext{
						Environment:  environment,
						Organization: org,
						Space:        space,
						Application:  appName,
					},
					Authorization: I.Authorization{
						Username: "TestUsername",
						Password: "TestPassword",
					},
				}
				controller.RunDeployment(deployment, response)

				Eventually(deployer.DeployCall.Received.Authorization.Username).Should(Equal("TestUsername"))
				Eventually(deployer.DeployCall.Received.Authorization.Password).Should(Equal("TestPassword"))
			})
		})

		Context("when SILENT_DEPLOY_ENVIRONMENT is true", func() {
			It("channel resolves true when no errors occur", func() {

				os.Setenv("SILENT_DEPLOY_ENVIRONMENT", environment)
				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				response := &bytes.Buffer{}

				deployment := &I.Deployment{
					Body: &[]byte{},
					Type: I.DeploymentType{JSON: true},
					CFContext: I.CFContext{
						Environment:  environment,
						Organization: org,
						Space:        space,
						Application:  appName,
					},
				}
				deployResponse := controller.RunDeployment(deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(1))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.ContentType).Should(Equal(I.DeploymentType{JSON: true}))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})
			It("channel resolves when no errors occur", func() {

				os.Setenv("SILENT_DEPLOY_ENVIRONMENT", environment)
				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				silentDeployer.DeployCall.Returns.Error = errors.New("bork")
				silentDeployer.DeployCall.Returns.StatusCode = http.StatusInternalServerError

				response := &bytes.Buffer{}

				silentDeployUrl := server.URL + "/v1/apps/" + os.Getenv("SILENT_DEPLOY_ENVIRONMENT")
				os.Setenv("SILENT_DEPLOY_URL", silentDeployUrl)

				deployment := &I.Deployment{
					Body: &[]byte{},
					Type: I.DeploymentType{JSON: true},
					CFContext: I.CFContext{
						Environment:  environment,
						Organization: org,
						Space:        space,
						Application:  appName,
					},
				}
				deployResponse := controller.RunDeployment(deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(1))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.ContentType).Should(Equal(I.DeploymentType{JSON: true}))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})
		})

		Context("when called", func() {
			It("creates a pusher creator", func() {
				bodyString := "body string"
				bodyBytes := []byte(bodyString)
				deployment := I.Deployment{
					Body: &bodyBytes,
					Type: I.DeploymentType{JSON: true},
					CFContext: I.CFContext{
						Environment:  environment,
						Organization: org,
						Space:        space,
						Application:  appName,
					},
				}
				pusherCreatorFactory := &mocks.PusherCreatorFactory{}
				response := &bytes.Buffer{}
				controller = &Controller{
					Deployer:             deployer,
					SilentDeployer:       silentDeployer,
					Log:                  logger.DefaultLogger(GinkgoWriter, logging.DEBUG, "api_test"),
					PusherCreatorFactory: pusherCreatorFactory,
				}

				controller.RunDeployment(&deployment, response)
				Eventually(pusherCreatorFactory.PusherCreatorCall.Called).Should(Equal(true))

			})
			It("Provides body for pusher creator", func() {
				bodyByte := []byte("body string")
				deployment := I.Deployment{
					Body: &bodyByte,
					Type: I.DeploymentType{JSON: true},
					CFContext: I.CFContext{
						Environment:  environment,
						Organization: org,
						Space:        space,
						Application:  appName,
					},
				}

				pusherCreatorFactory := &mocks.PusherCreatorFactory{}

				response := &bytes.Buffer{}
				controller = &Controller{
					Deployer:             deployer,
					SilentDeployer:       silentDeployer,
					Log:                  logger.DefaultLogger(GinkgoWriter, logging.DEBUG, "api_test"),
					PusherCreatorFactory: pusherCreatorFactory,
				}

				controller.RunDeployment(&deployment, response)
				returnedBody, _ := ioutil.ReadAll(pusherCreatorFactory.PusherCreatorCall.Received.Body)
				Eventually(returnedBody).Should(Equal(bodyByte))
			})
		})

	})
	Describe("StopDeployment", func() {
		Context("when verbose deployer is called", func() {
			It("channel resolves when no errors occur", func() {

				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				response := &bytes.Buffer{}

				deployment := &I.Deployment{
					Body: &[]byte{},
					Type: I.DeploymentType{JSON: true},
					CFContext: I.CFContext{
						Environment:  environment,
						Organization: org,
						Space:        space,
						Application:  appName,
					},
				}
				deployResponse := controller.RunDeployment(deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.ContentType).Should(Equal(I.DeploymentType{JSON: true}))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})

			It("channel resolves when errors occur", func() {

				deployer.DeployCall.Returns.Error = errors.New("bork")
				deployer.DeployCall.Returns.StatusCode = http.StatusInternalServerError
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				response := &bytes.Buffer{}

				deployment := &I.Deployment{
					Body: &[]byte{},
					Type: I.DeploymentType{JSON: true},
					CFContext: I.CFContext{
						Environment:  environment,
						Organization: org,
						Space:        space,
						Application:  appName,
					},
				}
				deployResponse := controller.RunDeployment(deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusInternalServerError))
				Eventually(deployResponse.Error.Error()).Should(Equal("bork"))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.ContentType).Should(Equal(I.DeploymentType{JSON: true}))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})

			It("does not set the basic auth header if no credentials are passed", func() {
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				response := &bytes.Buffer{}

				deployment := &I.Deployment{
					Body: &[]byte{},
					Type: I.DeploymentType{JSON: true},
					CFContext: I.CFContext{
						Environment:  environment,
						Organization: org,
						Space:        space,
						Application:  appName,
					},
					Authorization: I.Authorization{
						Username: "",
						Password: "",
					},
				}
				controller.RunDeployment(deployment, response)

				Eventually(deployer.DeployCall.Received.Authorization.Username).Should(Equal(""))
				Eventually(deployer.DeployCall.Received.Authorization.Password).Should(Equal(""))
			})

			It("sets the basic auth header if credentials are passed", func() {
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				response := &bytes.Buffer{}

				deployment := &I.Deployment{
					Body: &[]byte{},
					Type: I.DeploymentType{JSON: true},
					CFContext: I.CFContext{
						Environment:  environment,
						Organization: org,
						Space:        space,
						Application:  appName,
					},
					Authorization: I.Authorization{
						Username: "TestUsername",
						Password: "TestPassword",
					},
				}
				controller.RunDeployment(deployment, response)

				Eventually(deployer.DeployCall.Received.Authorization.Username).Should(Equal("TestUsername"))
				Eventually(deployer.DeployCall.Received.Authorization.Password).Should(Equal("TestPassword"))
			})
		})

		Context("when SILENT_DEPLOY_ENVIRONMENT is true", func() {
			It("channel resolves true when no errors occur", func() {

				os.Setenv("SILENT_DEPLOY_ENVIRONMENT", environment)
				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				response := &bytes.Buffer{}

				deployment := &I.Deployment{
					Body: &[]byte{},
					Type: I.DeploymentType{JSON: true},
					CFContext: I.CFContext{
						Environment:  environment,
						Organization: org,
						Space:        space,
						Application:  appName,
					},
				}
				deployResponse := controller.RunDeployment(deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(1))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.ContentType).Should(Equal(I.DeploymentType{JSON: true}))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})
			It("channel resolves when no errors occur", func() {

				os.Setenv("SILENT_DEPLOY_ENVIRONMENT", environment)
				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				silentDeployer.DeployCall.Returns.Error = errors.New("bork")
				silentDeployer.DeployCall.Returns.StatusCode = http.StatusInternalServerError

				response := &bytes.Buffer{}

				silentDeployUrl := server.URL + "/v1/apps/" + os.Getenv("SILENT_DEPLOY_ENVIRONMENT")
				os.Setenv("SILENT_DEPLOY_URL", silentDeployUrl)

				deployment := &I.Deployment{
					Body: &[]byte{},
					Type: I.DeploymentType{JSON: true},
					CFContext: I.CFContext{
						Environment:  environment,
						Organization: org,
						Space:        space,
						Application:  appName,
					},
				}
				deployResponse := controller.RunDeployment(deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(1))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.ContentType).Should(Equal(I.DeploymentType{JSON: true}))
				Eventually(deployer.DeployCall.Received.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})
		})
	})
})
