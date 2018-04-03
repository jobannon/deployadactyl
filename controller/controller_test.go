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
	"github.com/compozed/deployadactyl/constants"
	. "github.com/compozed/deployadactyl/controller"
	D "github.com/compozed/deployadactyl/controller/deployer"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/controller/deployer/error_finder"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/compozed/deployadactyl/state/stop"
	"github.com/compozed/deployadactyl/structs"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/op/go-logging"
	"reflect"
)

var _ = Describe("Controller", func() {

	var (
		deployer           *mocks.Deployer
		silentDeployer     *mocks.Deployer
		pushManagerFactory *mocks.PushManagerFactory
		stopManagerFactory *mocks.StopManagerFactory
		eventManager       *mocks.EventManager
		errorFinder        *mocks.ErrorFinder
		controller         *Controller
		deployment         I.Deployment
		logBuffer          *Buffer

		appName     string
		environment string
		org         string
		space       string
		byteBody    []byte
		server      *httptest.Server
		response    *bytes.Buffer
	)

	BeforeEach(func() {
		logBuffer = NewBuffer()
		appName = "appName-" + randomizer.StringRunes(10)
		environment = "environment-" + randomizer.StringRunes(10)
		org = "org-" + randomizer.StringRunes(10)
		space = "non-prod"

		eventManager = &mocks.EventManager{}
		deployer = &mocks.Deployer{}
		silentDeployer = &mocks.Deployer{}
		pushManagerFactory = &mocks.PushManagerFactory{}

		stopManagerFactory = &mocks.StopManagerFactory{}
		errorFinder = &mocks.ErrorFinder{}
		controller = &Controller{
			Deployer:           deployer,
			SilentDeployer:     silentDeployer,
			Log:                logger.DefaultLogger(logBuffer, logging.DEBUG, "api_test"),
			PushManagerFactory: pushManagerFactory,
			StopManagerFactory: stopManagerFactory,
			EventManager:       eventManager,
			Config:             config.Config{},
			ErrorFinder:        errorFinder,
		}
		environments := map[string]structs.Environment{}
		environments[environment] = structs.Environment{}
		controller.Config.Environments = environments
		bodyByte := []byte("{}")
		response = &bytes.Buffer{}

		deployment = I.Deployment{
			Body:          &bodyByte,
			Type:          I.DeploymentType{},
			CFContext:     I.CFContext{},
			Authorization: I.Authorization{},
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

				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "deploy success"

				router.ServeHTTP(resp, req)

				Eventually(resp.Code).Should(Equal(http.StatusOK))
				Eventually(resp.Body).Should(ContainSubstring("deploy success"))

				Eventually(deployer.DeployCall.Received.DeploymentInfo.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.AppName).Should(Equal(appName))
			})

			It("does not run silent deploy when environment other than non-prop", func() {
				foundationURL = fmt.Sprintf("/v2/deploy/%s/%s/%s/%s", environment, org, "not-non-prod", appName)

				req, err := http.NewRequest("POST", foundationURL, jsonBuffer)
				req.Header.Set("Content-Type", "application/zip")

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
				req.Header.Set("Content-Type", "application/zip")

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
				req.Header.Set("Content-Type", "application/zip")

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
				deployment.Type.ZIP = true
				deployResponse := controller.RunDeployment(deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.DeploymentInfo.Username).Should(Equal(deployment.Authorization.Username))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.Password).Should(Equal(deployment.Authorization.Password))
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
				deployment.Type.ZIP = true

				deployResponse := controller.RunDeployment(deployment, response)
				receivedBody, _ := ioutil.ReadAll(deployer.DeployCall.Received.DeploymentInfo.Body)
				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(receivedBody).Should(Equal(*deployment.Body))
			})

			It("channel resolves when no errors occur", func() {
				deployment.CFContext.Environment = environment
				deployment.CFContext.Organization = org
				deployment.CFContext.Space = space
				deployment.CFContext.Application = appName
				deployment.Type.ZIP = true

				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				deployResponse := controller.RunDeployment(&deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.DeploymentInfo.ContentType).Should(Equal("ZIP"))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})

			It("channel resolves when errors occur", func() {
				deployment.CFContext.Environment = environment
				deployment.CFContext.Organization = org
				deployment.CFContext.Space = space
				deployment.CFContext.Application = appName
				deployment.Type.ZIP = true

				deployer.DeployCall.Returns.Error = errors.New("bork")
				deployer.DeployCall.Returns.StatusCode = http.StatusInternalServerError
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				deployResponse := controller.RunDeployment(&deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(0))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusInternalServerError))
				Eventually(deployResponse.Error.Error()).Should(Equal("bork"))

				Eventually(deployer.DeployCall.Received.DeploymentInfo.ContentType).Should(Equal("ZIP"))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})

			It("does not set the basic auth header if no credentials are passed", func() {
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				response := &bytes.Buffer{}

				deployment := &I.Deployment{
					Body: &[]byte{},
					Type: I.DeploymentType{ZIP: true},
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

				Eventually(deployer.DeployCall.Received.DeploymentInfo.Username).Should(Equal(""))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.Password).Should(Equal(""))
			})

			It("sets the basic auth header if credentials are passed", func() {
				deployment.CFContext.Environment = environment
				deployment.Type.ZIP = true

				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				deployment.Authorization = I.Authorization{
					Username: "TestUsername",
					Password: "TestPassword",
				}

				controller.RunDeployment(&deployment, response)

				Eventually(deployer.DeployCall.Received.DeploymentInfo.Username).Should(Equal("TestUsername"))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.Password).Should(Equal("TestPassword"))
			})
		})

		Context("when SILENT_DEPLOY_ENVIRONMENT is true", func() {
			It("channel resolves true when no errors occur", func() {
				deployment.CFContext.Environment = environment
				deployment.CFContext.Organization = org
				deployment.CFContext.Space = space
				deployment.CFContext.Application = appName
				deployment.Type.ZIP = true

				os.Setenv("SILENT_DEPLOY_ENVIRONMENT", environment)
				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				deployResponse := controller.RunDeployment(&deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(1))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.DeploymentInfo.ContentType).Should(Equal("ZIP"))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})
			It("channel resolves when no errors occur", func() {
				deployment.CFContext.Environment = environment
				deployment.CFContext.Organization = org
				deployment.CFContext.Space = space
				deployment.CFContext.Application = appName
				deployment.Type.ZIP = true

				os.Setenv("SILENT_DEPLOY_ENVIRONMENT", environment)
				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "little-timmy-env.zip"

				silentDeployer.DeployCall.Returns.Error = errors.New("bork")
				silentDeployer.DeployCall.Returns.StatusCode = http.StatusInternalServerError

				silentDeployUrl := server.URL + "/v1/apps/" + os.Getenv("SILENT_DEPLOY_ENVIRONMENT")
				os.Setenv("SILENT_DEPLOY_URL", silentDeployUrl)

				deployResponse := controller.RunDeployment(&deployment, response)

				Eventually(deployer.DeployCall.Called).Should(Equal(1))
				Eventually(silentDeployer.DeployCall.Called).Should(Equal(1))

				Eventually(deployResponse.StatusCode).Should(Equal(http.StatusOK))

				Eventually(deployer.DeployCall.Received.DeploymentInfo.ContentType).Should(Equal("ZIP"))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.Environment).Should(Equal(environment))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.Org).Should(Equal(org))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.Space).Should(Equal(space))
				Eventually(deployer.DeployCall.Received.DeploymentInfo.AppName).Should(Equal(appName))

				ret, _ := ioutil.ReadAll(response)
				Eventually(string(ret)).Should(Equal("little-timmy-env.zip"))
			})
		})

		Context("when called", func() {
			It("logs building deploymentInfo", func() {
				deployment.CFContext.Environment = environment

				controller.RunDeployment(&deployment, response)
				Eventually(logBuffer).Should(Say("building deploymentInfo"))
			})
			It("creates a pusher creator", func() {
				deployment.CFContext.Environment = environment
				deployment.Type.ZIP = true

				controller.RunDeployment(&deployment, response)
				Eventually(pushManagerFactory.PusherCreatorCall.Called).Should(Equal(true))

			})
			It("Provides body for pusher creator", func() {
				bodyByte := []byte("body string")
				deployment.CFContext.Environment = environment
				deployment.Body = &bodyByte
				deployment.Type.ZIP = true

				controller.RunDeployment(&deployment, response)
				returnedBody, _ := ioutil.ReadAll(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.RequestBody)
				Eventually(returnedBody).Should(Equal(bodyByte))
			})
			It("Provides response for pusher creator", func() {
				deployment.CFContext.Environment = environment
				deployment.Type.ZIP = true

				response = bytes.NewBuffer([]byte("hello"))

				controller.RunDeployment(&deployment, response)
				returnedResponse, _ := ioutil.ReadAll(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.Response)
				Eventually(returnedResponse).Should(Equal([]byte("hello")))
			})
			Context("when type is JSON", func() {
				It("gets the artifact url from the request", func() {
					bodyByte := []byte("{\"artifact_url\": \"the artifact url\"}")
					deployment.Body = &bodyByte
					deployment.CFContext.Environment = environment
					deployment.Type.JSON = true

					controller.RunDeployment(&deployment, response)
					Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.ArtifactURL).Should(Equal("the artifact url"))
				})
				It("gets the manifest from the request", func() {
					bodyByte := []byte("{\"artifact_url\": \"the artifact url\", \"manifest\": \"the manifest\"}")
					deployment.Body = &bodyByte
					deployment.CFContext.Environment = environment
					deployment.Type.JSON = true

					controller.RunDeployment(&deployment, response)
					Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Manifest).Should(Equal("the manifest"))
				})
				It("gets the data from the request", func() {
					bodyByte := []byte("{\"artifact_url\": \"the artifact url\", \"data\": {\"avalue\": \"the data\"}}")
					deployment.Body = &bodyByte
					deployment.CFContext.Environment = environment
					deployment.Type.JSON = true

					controller.RunDeployment(&deployment, response)
					Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Data["avalue"]).Should(Equal("the data"))
				})
			})
			Context("the deployment info", func() {
				Context("when environment does not exist", func() {
					It("returns an error with StatusInternalServerError", func() {
						deployment.CFContext.Environment = "bad env"
						deployment.Type.ZIP = true

						deploymentResponse := controller.RunDeployment(&deployment, response)
						Eventually(deploymentResponse.Error).Should(HaveOccurred())
						Eventually(reflect.TypeOf(deploymentResponse.Error)).Should(Equal(reflect.TypeOf(D.EnvironmentNotFoundError{})))
					})
				})
				Context("when environment exists", func() {
					Context("when Authorization doesn't have values", func() {
						Context("and authentication is not required", func() {
							It("returns username and password from the config", func() {
								deployment.CFContext.Environment = environment
								deployment.Type.ZIP = true

								deployment.Authorization.Username = ""
								deployment.Authorization.Password = ""
								controller.Config.Username = "username-" + randomizer.StringRunes(10)
								controller.Config.Password = "password-" + randomizer.StringRunes(10)

								controller.RunDeployment(&deployment, response)

								Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Username).Should(Equal(controller.Config.Username))
								Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Password).Should(Equal(controller.Config.Password))
							})
						})
						Context("and authentication is required", func() {
							It("returns an error", func() {
								deployment.CFContext.Environment = environment
								deployment.Type.ZIP = true

								deployment.Authorization.Username = ""
								deployment.Authorization.Password = ""

								controller.Config.Environments[environment] = structs.Environment{
									Authenticate: true,
								}

								deploymentResponse := controller.RunDeployment(&deployment, response)

								Eventually(deploymentResponse.Error).Should(HaveOccurred())
								Eventually(deploymentResponse.Error.Error()).Should(Equal("basic auth header not found"))
							})
						})
					})
					Context("when Authorization has values", func() {
						It("logs checking auth", func() {
							deployment.CFContext.Environment = environment
							deployment.Type.ZIP = true

							controller.RunDeployment(&deployment, response)

							Eventually(logBuffer).Should(Say("checking for basic auth"))
						})
						It("returns username and password from the authorization", func() {
							deployment.CFContext.Environment = environment
							deployment.Type.ZIP = true

							deployment.Authorization.Username = "username-" + randomizer.StringRunes(10)
							deployment.Authorization.Password = "password-" + randomizer.StringRunes(10)

							controller.RunDeployment(&deployment, response)

							Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Username).Should(Equal(deployment.Authorization.Username))
							Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Password).Should(Equal(deployment.Authorization.Password))
						})
					})
					It("has the correct org, space ,appname, env, uuid", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						deployment.CFContext.Organization = org
						deployment.CFContext.Space = space
						deployment.CFContext.Application = appName
						deployment.CFContext.Environment = environment

						controller.RunDeployment(&deployment, response)

						Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Org).Should(Equal(org))
						Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Space).Should(Equal(space))
						Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.AppName).Should(Equal(appName))
						Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Environment).Should(Equal(environment))
					})
					It("has the correct JSON content type", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.JSON = true
						bodyByte := []byte(`{"artifact_url": "xyz"}`)
						deployment.Body = &bodyByte

						controller.RunDeployment(&deployment, response)

						Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.ContentType).Should(Equal("JSON"))
					})
					It("has the correct ZIP content type", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.ContentType).Should(Equal("ZIP"))
					})
					It("has the correct body", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true
						bodyByte := []byte(`{"artifact_url": "xyz"}`)
						deployment.Body = &bodyByte

						controller.RunDeployment(&deployment, response)

						returnedBody, _ := ioutil.ReadAll(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Body)
						Eventually(string(returnedBody)).Should(Equal(string(bodyByte)))
					})

					Context("when contentType is neither", func() {
						It("returns an error", func() {
							deployment.CFContext.Environment = environment

							deployResponse := controller.RunDeployment(&deployment, response)

							Eventually(reflect.TypeOf(deployResponse.Error)).Should(Equal(reflect.TypeOf(D.InvalidContentTypeError{})))
						})
					})

					Context("when uuid is not provided", func() {
						It("creates a new uuid", func() {
							deployment.CFContext.Environment = environment
							deployment.Type.ZIP = true

							controller.RunDeployment(&deployment, response)

							Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.UUID).ShouldNot(BeEmpty())
						})
					})
					Context("when uuid is provided", func() {
						It("uses the provided uuid", func() {
							deployment.CFContext.Environment = environment
							uuid := randomizer.StringRunes(10)
							deployment.CFContext.UUID = uuid
							deployment.Type.ZIP = true

							controller.RunDeployment(&deployment, response)

							Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.UUID).Should(Equal(uuid))

						})
					})
					It("has the correct domain and skipssl", func() {
						deployment.CFContext.Environment = environment
						domain := "domain-" + randomizer.StringRunes(10)
						deployment.Authorization.Username = ""
						deployment.Authorization.Password = ""
						deployment.Type.ZIP = true

						controller.Config.Environments[environment] = structs.Environment{
							Domain:  domain,
							SkipSSL: true,
						}

						controller.RunDeployment(&deployment, response)

						Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.Domain).Should(Equal(domain))
						Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.SkipSSL).Should(BeTrue())
					})
					It("has correct custom parameters", func() {

						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						customParams := make(map[string]interface{})
						customParams["param1"] = "value1"
						customParams["param2"] = "value2"

						controller.Config.Environments[environment] = structs.Environment{
							CustomParams: customParams,
						}

						controller.RunDeployment(&deployment, response)

						Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.CustomParams["param1"]).Should(Equal("value1"))
						Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.CustomParams["param2"]).Should(Equal("value2"))

					})
					It("is passed to the pusher creator", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo).ShouldNot(BeNil())
					})

					It("correctly extracts artifact url from body", func() {
						artifactURL := "artifactURL-" + randomizer.StringRunes(10)
						bodyByte := []byte(fmt.Sprintf(`{"artifact_url": "%s"}`, artifactURL))

						deployment.CFContext.Environment = environment
						deployment.Body = &bodyByte
						deployment.Type.JSON = true

						controller.RunDeployment(&deployment, response)

						Eventually(pushManagerFactory.PusherCreatorCall.Received.DeployEventData.DeploymentInfo.ArtifactURL).Should(Equal(artifactURL))
					})
					Context("if artifact url isn't provided in body", func() {
						It("returns an error", func() {
							bodyByte := []byte("{}")

							deployment.CFContext.Environment = environment
							deployment.Body = &bodyByte
							deployment.Type.JSON = true

							deploymentResponse := controller.RunDeployment(&deployment, response)

							Eventually(deploymentResponse.Error).ShouldNot(BeNil())
							Eventually(deploymentResponse.Error.Error()).Should(ContainSubstring("The following properties are missing: artifact_url"))
						})
					})
					Context("if body is invalid", func() {
						It("returns an error", func() {
							bodyByte := []byte("")

							deployment.CFContext.Environment = environment
							deployment.Body = &bodyByte
							deployment.Type.JSON = true

							deploymentResponse := controller.RunDeployment(&deployment, response)

							Eventually(deploymentResponse.Error).ShouldNot(BeNil())
							Eventually(deploymentResponse.Error.Error()).Should(ContainSubstring("EOF"))
						})
					})
					It("logs a start event", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						Eventually(logBuffer).Should(Say("emitting a deploy.start event"))
					})
					It("emits a start event", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						Expect(eventManager.EmitCall.Received.Events[0].Type).Should(Equal(constants.DeployStartEvent))
					})
					Context("when start emit fails", func() {
						It("returns error", func() {
							deployment.CFContext.Environment = environment
							deployment.Type.ZIP = true

							eventManager.EmitCall.Returns.Error = []error{errors.New("a test error")}

							deploymentResponse := controller.RunDeployment(&deployment, response)

							Expect(reflect.TypeOf(deploymentResponse.Error)).Should(Equal(reflect.TypeOf(D.EventError{})))
						})
					})
					It("passes populated deploymentInfo to DeployStartEvent event", func() {
						deployment.CFContext.Environment = environment
						deployment.CFContext.Application = appName
						deployment.CFContext.Space = space
						deployment.CFContext.Organization = org
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						deploymentInfo := eventManager.EmitCall.Received.Events[0].Data.(*structs.DeployEventData).DeploymentInfo
						Expect(deploymentInfo.AppName).To(Equal(appName))
						Expect(deploymentInfo.Org).To(Equal(org))
						Expect(deploymentInfo.Space).To(Equal(space))
						Expect(deploymentInfo.UUID).ToNot(BeNil())
					})
					It("emits a finished event", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)
						//
						Expect(eventManager.EmitCall.Received.Events[2].Type).Should(Equal(constants.DeployFinishEvent))
					})
					Context("when finished emit fails", func() {
						It("returns error", func() {
							deployment.CFContext.Environment = environment
							deployment.Type.ZIP = true

							eventManager.EmitCall.Returns.Error = []error{nil, nil, errors.New("a test error")}

							deploymentResponse := controller.RunDeployment(&deployment, response)

							Expect(reflect.TypeOf(deploymentResponse.Error)).Should(Equal(reflect.TypeOf(bluegreen.FinishDeployError{})))
						})
					})
					It("emits a success event", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)
						Expect(eventManager.EmitCall.Received.Events[1].Type).Should(Equal(constants.DeploySuccessEvent))
					})
					Context("when emit successOrFailure fails", func() {
						It("logs an error", func() {
							deployment.CFContext.Environment = environment
							deployment.Type.ZIP = true

							eventManager.EmitCall.Returns.Error = []error{nil, errors.New("a test error"), nil}

							controller.RunDeployment(&deployment, response)
							Eventually(logBuffer).Should(Say("an error occurred when emitting a deploy.success event"))
						})
					})
					It("emits a failure event", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						eventManager.EmitCall.Returns.Error = []error{errors.New("a test error"), nil, nil}

						controller.RunDeployment(&deployment, response)
						Expect(eventManager.EmitCall.Received.Events[1].Type).Should(Equal(constants.DeployFailureEvent))
					})
					It("logs emitting an event", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)
						Eventually(logBuffer).Should(Say("emitting a deploy.success event"))
					})
					It("prints found errors to the response", func() {
						deployment.CFContext.Environment = environment
						deployment.Type.ZIP = true

						eventManager.EmitCall.Returns.Error = []error{errors.New("a test error"), nil, nil}

						retError := error_finder.CreateLogMatchedError("a description", []string{"some details"}, "a solution", "a code")
						errorFinder.FindErrorsCall.Returns.Errors = []I.LogMatchedError{retError}

						controller.RunDeployment(&deployment, response)
						responseBytes, _ := ioutil.ReadAll(response)
						Eventually(string(responseBytes)).Should(ContainSubstring("The following error was found in the above logs: a description"))
						Eventually(string(responseBytes)).Should(ContainSubstring("Error: some details"))
						Eventually(string(responseBytes)).Should(ContainSubstring("Potential solution: a solution"))
					})
					It("passes populated deploymentInfo to DeploySuccessEvent event", func() {
						deployment.CFContext.Environment = environment
						deployment.CFContext.Application = appName
						deployment.CFContext.Space = space
						deployment.CFContext.Organization = org
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						deploymentInfo := eventManager.EmitCall.Received.Events[1].Data.(*structs.DeployEventData).DeploymentInfo
						Expect(deploymentInfo.AppName).To(Equal(appName))
						Expect(deploymentInfo.Org).To(Equal(org))
						Expect(deploymentInfo.Space).To(Equal(space))
						Expect(deploymentInfo.UUID).ToNot(BeNil())
					})
					It("passes populated deploymentInfo to DeployFinishEvent event", func() {
						deployment.CFContext.Environment = environment
						deployment.CFContext.Application = appName
						deployment.CFContext.Space = space
						deployment.CFContext.Organization = org
						deployment.Type.ZIP = true

						controller.RunDeployment(&deployment, response)

						deploymentInfo := eventManager.EmitCall.Received.Events[2].Data.(*structs.DeployEventData).DeploymentInfo
						Expect(deploymentInfo.AppName).To(Equal(appName))
						Expect(deploymentInfo.Org).To(Equal(org))
						Expect(deploymentInfo.Space).To(Equal(space))
						Expect(deploymentInfo.UUID).ToNot(BeNil())
					})
				})
			})

		})

	})
	Describe("StopDeployment", func() {
		Context("When UUID is not provided", func() {
			It("Should populate UUID", func() {

				deployment := &I.Deployment{
					CFContext: I.CFContext{
						Environment: environment,
					}}
				response := bytes.NewBuffer([]byte{})
				deploymentResponse := controller.StopDeployment(deployment, nil, response)

				Expect(deploymentResponse.DeploymentInfo.UUID).ShouldNot(BeEmpty())
			})
		})
		Context("When UUID is provided", func() {
			It("Should return UUID", func() {

				deployment := &I.Deployment{
					CFContext: I.CFContext{
						UUID:        "myUUID",
						Environment: environment,
					},
				}
				response := bytes.NewBuffer([]byte{})
				deploymentResponse := controller.StopDeployment(deployment, nil, response)

				Expect(deploymentResponse.DeploymentInfo.UUID).ShouldNot(BeEmpty())
				Expect(deploymentResponse.DeploymentInfo.UUID).Should(Equal("myUUID"))

			})
		})
		It("Should return org, space, appname, and environment when provided", func() {

			deployment := &I.Deployment{
				CFContext: I.CFContext{
					Organization: "myOrg",
					Space:        "mySpace",
					Application:  "myApp",
					Environment:  environment,
				},
			}
			response := bytes.NewBuffer([]byte{})
			deploymentResponse := controller.StopDeployment(deployment, nil, response)

			Expect(deploymentResponse.DeploymentInfo.Org).Should(Equal("myOrg"))
			Expect(deploymentResponse.DeploymentInfo.Environment).Should(Equal(environment))
			Expect(deploymentResponse.DeploymentInfo.Space).Should(Equal("mySpace"))
			Expect(deploymentResponse.DeploymentInfo.AppName).Should(Equal("myApp"))

		})

		It("Should log start of process", func() {

			deployment := &I.Deployment{
				CFContext: I.CFContext{
					Application: "myApp",
					Environment: environment,
				},
			}

			response := bytes.NewBuffer([]byte{})
			deploymentResponse := controller.StopDeployment(deployment, nil, response)

			Expect(logBuffer).Should(Say(fmt.Sprintf("Preparing to stop %s with UUID %s", "myApp", deploymentResponse.DeploymentInfo.UUID)))

		})
		Context("When StopStartEvent succeeds", func() {
			It("should emit a StopStarteEvent", func() {
				deployment := &I.Deployment{
					CFContext: I.CFContext{
						Organization: "myOrg",
						Space:        "mySpace",
						Application:  "myApp",
						Environment:  environment,
						UUID:         "myUUID",
					},
				}
				data := make(map[string]interface{})
				data["mykey"] = "first value"
				controller.StopDeployment(deployment, data, response)

				Expect(reflect.TypeOf(eventManager.EmitEventCall.Received.Events[0])).Should(Equal(reflect.TypeOf(stop.StopStartedEvent{})))
				stopEvent := eventManager.EmitEventCall.Received.Events[0].(stop.StopStartedEvent)
				Expect(stopEvent.CFContext.UUID).Should(Equal("myUUID"))
				Expect(stopEvent.CFContext.Space).Should(Equal("mySpace"))
				Expect(stopEvent.CFContext.Application).Should(Equal("myApp"))
				Expect(stopEvent.CFContext.Environment).Should(Equal(environment))
				Expect(stopEvent.CFContext.Organization).Should(Equal("myOrg"))
				Expect(stopEvent.Data).Should(Equal(data))

			})
		})
		Context("When StopStartEvent fails", func() {
			It("should return error", func() {
				eventManager.EmitEventCall.Returns.Error = append(eventManager.EmitEventCall.Returns.Error, errors.New("anything"))

				deployment := &I.Deployment{}
				deployResponse := controller.StopDeployment(deployment, nil, response)

				Expect(deployResponse.StatusCode).Should(Equal(http.StatusInternalServerError))
				Expect(reflect.TypeOf(deployResponse.Error)).Should(Equal(reflect.TypeOf(D.EventError{})))

			})
		})

		Context("When environment does not exist", func() {
			It("Should return error", func() {

				deployment := &I.Deployment{
					CFContext: I.CFContext{
						Environment: "bad environment",
					}}
				response := bytes.NewBuffer([]byte{})
				deploymentResponse := controller.StopDeployment(deployment, nil, response)

				Expect(reflect.TypeOf(deploymentResponse.Error)).Should(Equal(reflect.TypeOf(D.EnvironmentNotFoundError{})))
			})
		})
		Context("When environment exists", func() {
			It("Should return SkipSSL, CustomParams, and Domain", func() {

				controller.Config.Environments[environment] = structs.Environment{
					SkipSSL:      true,
					Domain:       "myDomain",
					CustomParams: make(map[string]interface{}),
				}
				controller.Config.Environments[environment].CustomParams["customName"] = "customParams"

				deployment := &I.Deployment{
					CFContext: I.CFContext{
						Environment: environment,
					}}

				response := bytes.NewBuffer([]byte{})
				deploymentResponse := controller.StopDeployment(deployment, nil, response)
				Expect(deploymentResponse.DeploymentInfo.Domain).Should(Equal("myDomain"))
				Expect(deploymentResponse.DeploymentInfo.SkipSSL).Should(Equal(true))
				Expect(deploymentResponse.DeploymentInfo.CustomParams["customName"]).Should(Equal("customParams"))
			})
		})
		Context("When auth does not exist", func() {
			Context("When environment authenticate is true", func() {
				It("Should return error", func() {
					controller.Config.Environments[environment] = structs.Environment{
						Authenticate: true,
					}
					deployment := &I.Deployment{
						CFContext: I.CFContext{
							Environment: environment,
						}}
					response := bytes.NewBuffer([]byte{})
					deploymentResponse := controller.StopDeployment(deployment, nil, response)

					Expect(reflect.TypeOf(deploymentResponse.Error)).Should(Equal(reflect.TypeOf(D.BasicAuthError{})))
				})
			})
			Context("When environment authenticate is false", func() {
				It("Should username and password using the config", func() {
					controller.Config.Username = "username"
					controller.Config.Password = "password"
					controller.Config.Environments[environment] = structs.Environment{
						Authenticate: false,
					}
					deployment := &I.Deployment{
						CFContext: I.CFContext{
							Environment: environment,
						}}
					response := bytes.NewBuffer([]byte{})
					deploymentResponse := controller.StopDeployment(deployment, nil, response)

					Expect(deploymentResponse.DeploymentInfo.Username).Should(Equal("username"))
					Expect(deploymentResponse.DeploymentInfo.Password).Should(Equal("password"))

				})
			})
		})
		Context("When auth is provided", func() {
			It("Should populate the deploymentInfo with the username and password", func() {
				deployment := &I.Deployment{
					Authorization: I.Authorization{
						Username: "myUser",
						Password: "myPassword",
					},
					CFContext: I.CFContext{
						Environment: environment,
					},
				}
				response := bytes.NewBuffer([]byte{})
				deploymentResponse := controller.StopDeployment(deployment, nil, response)
				Expect(deploymentResponse.DeploymentInfo.Username).Should(Equal("myUser"))
				Expect(deploymentResponse.DeploymentInfo.Password).Should(Equal("myPassword"))
			})
		})
		Context("When auth is provided", func() {
			It("Should populate the deploymentInfo with the username and password", func() {
				deployment := &I.Deployment{
					Authorization: I.Authorization{
						Username: "myUser",
						Password: "myPassword",
					},
					CFContext: I.CFContext{
						Environment: environment,
					},
				}
				response := bytes.NewBuffer([]byte{})
				deploymentResponse := controller.StopDeployment(deployment, nil, response)
				Expect(deploymentResponse.DeploymentInfo.Username).Should(Equal("myUser"))
				Expect(deploymentResponse.DeploymentInfo.Password).Should(Equal("myPassword"))
			})
		})

		Context("When data is provided", func() {
			It("should return deployment info with proper data", func() {
				data := map[string]interface{}{
					"user_id": "myuserid",
					"group":   "mygroup",
				}
				deployment := &I.Deployment{
					CFContext: I.CFContext{
						Environment: environment,
					},
				}
				response := bytes.NewBuffer([]byte{})
				deploymentResponse := controller.StopDeployment(deployment, data, response)
				Expect(deploymentResponse.DeploymentInfo.Data["user_id"]).Should(Equal("myuserid"))
				Expect(deploymentResponse.DeploymentInfo.Data["group"]).Should(Equal("mygroup"))

			})
		})
		It("should create stop manager", func() {

			deployment := &I.Deployment{
				Authorization: I.Authorization{
					Username: "myUser",
				},
				CFContext: I.CFContext{
					Environment: environment,
				},
			}
			response := bytes.NewBuffer([]byte{})
			controller.StopDeployment(deployment, nil, response)
			Expect(stopManagerFactory.StopManagerCall.Called).Should(Equal(true))
			Expect(stopManagerFactory.StopManagerCall.Received.DeployEventData.DeploymentInfo.Username).Should(Equal("myUser"))
		})
		It("should call deploy with the stop manager ", func() {
			manager := &mocks.StopManager{}
			stopManagerFactory.StopManagerCall.Returns.ActionCreater = manager
			deployment := &I.Deployment{
				CFContext: I.CFContext{
					Environment: environment,
				},
			}
			response := bytes.NewBuffer([]byte{})
			controller.StopDeployment(deployment, nil, response)
			Expect(deployer.DeployCall.Received.ActionCreator).Should(Equal(manager))
		})

		It("should call deploy with the stop manager ", func() {
			deployer.DeployCall.Returns.Error = errors.New("test error")
			deployer.DeployCall.Returns.StatusCode = http.StatusOK

			deployment := &I.Deployment{
				CFContext: I.CFContext{
					Environment: environment,
				},
			}
			response := bytes.NewBuffer([]byte{})
			deploymentResponse := controller.StopDeployment(deployment, nil, response)

			Expect(deploymentResponse.Error.Error()).Should(Equal("test error"))
			Expect(deploymentResponse.StatusCode).Should(Equal(http.StatusOK))

		})

		Context("when stop succeeds", func() {
			Context("if StopSuccessEvent succeeds", func() {
				It("should emit StopSuccessEvent", func() {
					deployment := &I.Deployment{
						CFContext: I.CFContext{
							Organization: "myOrg",
							Space:        "mySpace",
							Application:  "myApp",
							Environment:  environment,
							UUID:         "myUUID",
						},
						Authorization: I.Authorization{
							Username: "myUser",
							Password: "myPassword",
						},
					}
					response := bytes.NewBuffer([]byte{})
					data := make(map[string]interface{})
					data["mykey"] = "first value"
					controller.Config.Environments[environment] = structs.Environment{
						Name:         environment,
						Authenticate: true,
					}
					controller.StopDeployment(deployment, data, response)

					Expect(reflect.TypeOf(eventManager.EmitEventCall.Received.Events[1])).To(Equal(reflect.TypeOf(stop.StopSuccessEvent{})))
					stopSuccessEvent := eventManager.EmitEventCall.Received.Events[1].(stop.StopSuccessEvent)

					Expect(stopSuccessEvent.CFContext.UUID).Should(Equal("myUUID"))
					Expect(stopSuccessEvent.CFContext.Space).Should(Equal("mySpace"))
					Expect(stopSuccessEvent.CFContext.Application).Should(Equal("myApp"))
					Expect(stopSuccessEvent.CFContext.Environment).Should(Equal(environment))
					Expect(stopSuccessEvent.CFContext.Organization).Should(Equal("myOrg"))
					Expect(stopSuccessEvent.Authorization.Username).Should(Equal("myUser"))
					Expect(stopSuccessEvent.Authorization.Password).Should(Equal("myPassword"))
					Expect(stopSuccessEvent.Environment.Name).Should(Equal(environment))
					Expect(stopSuccessEvent.Data).Should(Equal(data))

				})
				It("should emit a StopStartedEvent", func() {
					deployment := &I.Deployment{
						CFContext: I.CFContext{
							Organization: "myOrg",
							Space:        "mySpace",
							Application:  "myApp",
							Environment:  environment,
							UUID:         "myUUID",
						},
					}
					data := make(map[string]interface{})
					data["mykey"] = "first value"
					controller.StopDeployment(deployment, data, response)

					Expect(reflect.TypeOf(eventManager.EmitEventCall.Received.Events[0])).Should(Equal(reflect.TypeOf(stop.StopStartedEvent{})))
					stopEvent := eventManager.EmitEventCall.Received.Events[0].(stop.StopStartedEvent)
					Expect(stopEvent.CFContext.UUID).Should(Equal("myUUID"))
					Expect(stopEvent.CFContext.Space).Should(Equal("mySpace"))
					Expect(stopEvent.CFContext.Application).Should(Equal("myApp"))
					Expect(stopEvent.CFContext.Environment).Should(Equal(environment))
					Expect(stopEvent.CFContext.Organization).Should(Equal("myOrg"))
					Expect(stopEvent.Data).Should(Equal(data))

				})
			})
			Context("if StopSuccessEvent fails", func() {
				It("should log the error", func() {
					eventManager.EmitEventCall.Returns.Error = []error{nil, errors.New("errors")}
					deployment := &I.Deployment{
						CFContext: I.CFContext{
							Environment: environment,
						},
					}
					response := bytes.NewBuffer([]byte{})
					controller.StopDeployment(deployment, nil, response)

					Eventually(logBuffer).Should(Say("an error occurred when emitting a StopSuccessEvent event: errors"))
				})
			})

		})
		Context("when stop fails", func() {
			It("print errors", func() {
				deployment := &I.Deployment{
					CFContext: I.CFContext{
						Environment: environment,
					},
				}
				deployer.DeployCall.Returns.Error = errors.New("deploy error")
				errorFinder.FindErrorsCall.Returns.Errors = []I.LogMatchedError{error_finder.CreateLogMatchedError("a test error", []string{"error 1", "error 2", "error 3"}, "error solution", "test code")}
				response := bytes.NewBuffer([]byte{})
				controller.StopDeployment(deployment, nil, response)
				Eventually(response).Should(ContainSubstring("Potential solution"))
			})
			It("should emit StopFailureEvent", func() {

				deployment := &I.Deployment{
					CFContext: I.CFContext{
						Organization: "myOrg",
						Space:        "mySpace",
						Application:  "myApp",
						Environment:  environment,
						UUID:         "myUUID",
					},
					Authorization: I.Authorization{
						Username: "myUser",
						Password: "myPassword",
					},
				}
				response := bytes.NewBuffer([]byte{})
				data := make(map[string]interface{})
				data["mykey"] = "first value"
				controller.Config.Environments[environment] = structs.Environment{
					Name:         environment,
					Authenticate: true,
				}
				deployer.DeployCall.Returns.Error = errors.New("deploy error")
				controller.StopDeployment(deployment, data, response)

				Expect(reflect.TypeOf(eventManager.EmitEventCall.Received.Events[1])).To(Equal(reflect.TypeOf(stop.StopFailureEvent{})))
				event := eventManager.EmitEventCall.Received.Events[1].(stop.StopFailureEvent)

				Expect(event.CFContext.UUID).Should(Equal("myUUID"))
				Expect(event.CFContext.Space).Should(Equal("mySpace"))
				Expect(event.CFContext.Application).Should(Equal("myApp"))
				Expect(event.CFContext.Environment).Should(Equal(environment))
				Expect(event.CFContext.Organization).Should(Equal("myOrg"))
				Expect(event.Authorization.Username).Should(Equal("myUser"))
				Expect(event.Authorization.Password).Should(Equal("myPassword"))
				Expect(event.Environment.Name).Should(Equal(environment))
				Expect(event.Data).Should(Equal(data))
				Expect(event.Error.Error()).Should(Equal("deploy error"))

			})
			Context("if StopFailureEvent fails", func() {
				It("should log the error", func() {
					eventManager.EmitEventCall.Returns.Error = []error{nil, errors.New("errors")}
					deployment := &I.Deployment{
						CFContext: I.CFContext{
							Environment: environment,
						},
					}
					deployer.DeployCall.Returns.Error = errors.New("deploy error")

					response := bytes.NewBuffer([]byte{})
					controller.StopDeployment(deployment, nil, response)

					Eventually(logBuffer).Should(Say("an error occurred when emitting a StopFailureEvent event: errors"))
				})
			})

		})
		Context("when stop finishes", func() {
			It("should log an emit StopFinish event", func() {
				deployment := &I.Deployment{
					CFContext: I.CFContext{
						Environment: environment,
					},
				}

				response := bytes.NewBuffer([]byte{})
				controller.StopDeployment(deployment, nil, response)

				Eventually(logBuffer).Should(Say("emitting a StopFinishedEvent"))
			})
			It("should emit StopFinishedEvent", func() {

				deployment := &I.Deployment{
					CFContext: I.CFContext{
						Organization: "myOrg",
						Space:        "mySpace",
						Application:  "myApp",
						Environment:  environment,
						UUID:         "myUUID",
					},
					Authorization: I.Authorization{
						Username: "myUser",
						Password: "myPassword",
					},
				}
				response := bytes.NewBuffer([]byte{})
				data := make(map[string]interface{})
				data["mykey"] = "first value"
				controller.Config.Environments[environment] = structs.Environment{
					Name:         environment,
					Authenticate: true,
				}
				controller.StopDeployment(deployment, data, response)

				Expect(reflect.TypeOf(eventManager.EmitEventCall.Received.Events[2])).To(Equal(reflect.TypeOf(stop.StopFinishedEvent{})))
				event := eventManager.EmitEventCall.Received.Events[2].(stop.StopFinishedEvent)

				Expect(event.CFContext.UUID).Should(Equal("myUUID"))
				Expect(event.CFContext.Space).Should(Equal("mySpace"))
				Expect(event.CFContext.Application).Should(Equal("myApp"))
				Expect(event.CFContext.Environment).Should(Equal(environment))
				Expect(event.CFContext.Organization).Should(Equal("myOrg"))
				Expect(event.Authorization.Username).Should(Equal("myUser"))
				Expect(event.Authorization.Password).Should(Equal("myPassword"))
				Expect(event.Environment.Name).Should(Equal(environment))
				Expect(event.Data).Should(Equal(data))

			})
		})
	})

})
