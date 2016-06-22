package deployadactyl_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/compozed/deployadactyl"
	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/compozed/deployadactyl/test/mocks"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/op/go-logging"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Deployadactyl", func() {

	var (
		deployadactyl  *Deployadactyl
		deployer       *mocks.Deployer
		deploymentInfo S.DeploymentInfo
		r              *gin.Engine
		resp           *httptest.ResponseRecorder
		req            *http.Request
		eventManager   *mocks.EventManager
		randomizerMock *mocks.Randomizer

		artifactUrl     string
		environment     string
		org             string
		space           string
		appName         string
		username        string
		password        string
		defaultUsername string
		defaultPassword string
		apiUrl          string
		uuid            string
	)

	BeforeEach(func() {
		deployer = &mocks.Deployer{}
		eventManager = &mocks.EventManager{}
		randomizerMock = &mocks.Randomizer{}
		envMap := map[string]config.Environment{}
		envMap["Test"] = config.Environment{Foundations: []string{"api1.example.com", "api2.example.com"}}
		envMap["Prod"] = config.Environment{Foundations: []string{"api3.example.com", "api4.example.com"}}

		artifactUrl = "artifactUrl-" + randomizer.StringRunes(10)
		environment = "environment-" + randomizer.StringRunes(10)
		org = "org-" + randomizer.StringRunes(10)
		space = "space-" + randomizer.StringRunes(10)
		appName = "appName-" + randomizer.StringRunes(10)
		username = "username-" + randomizer.StringRunes(10)
		password = "password-" + randomizer.StringRunes(10)
		defaultUsername = "defaultUsername-" + randomizer.StringRunes(10)
		defaultPassword = "defaultPassword-" + randomizer.StringRunes(10)
		uuid = "uuid-" + randomizer.StringRunes(123)

		c := config.Config{
			Username:     defaultUsername,
			Password:     defaultPassword,
			Environments: envMap,
		}

		deployadactyl = &Deployadactyl{
			Deployer:     deployer,
			Log:          logger.DefaultLogger(GinkgoWriter, logging.DEBUG, "api_test"),
			Config:       c,
			EventManager: eventManager,
			Randomizer:   randomizerMock,
		}

		apiUrl = fmt.Sprintf("/v1/apps/%s/%s/%s/%s",
			environment,
			org,
			space,
			appName,
		)

		deploymentInfo = S.DeploymentInfo{
			ArtifactURL: artifactUrl,
			Username:    username,
			Password:    password,
			Environment: environment,
			Org:         org,
			Space:       space,
			AppName:     appName,
			UUID:        uuid,
		}

		deployadactyl.Config.Environments[environment] = config.Environment{Authenticate: true}

		eventManager.On("Emit", mock.Anything).Return(nil)
		randomizerMock.On("StringRunes", mock.Anything).Return(uuid)

		r = gin.New()
		resp = httptest.NewRecorder()
	})

	AfterEach(func() {
		Expect(deployer.AssertExpectations(GinkgoT())).To(BeTrue())
	})

	Describe("Deploy", func() {
		BeforeEach(func() {
			r.POST("/v1/apps/:environment/:org/:space/:appName", deployadactyl.Deploy)
		})

		Context("when the request is missing properties", func() {
			It("returns an error", func() {
				jsonBuffer := bytes.NewBufferString("{}")

				req, err := http.NewRequest("POST", "/v1/apps/someEnv/someOrg/someSpace/someApp", jsonBuffer)
				Expect(err).ToNot(HaveOccurred())

				req.Header.Set("Content-Type", "application/json")

				req.SetBasicAuth(username, password)

				r.ServeHTTP(resp, req)

				Expect(resp.Code).To(Equal(500))
				Expect(resp.Body.String()).To(Equal("The following properties are missing: artifact_url"))
			})
		})

		Describe("Authentication", func() {
			Context("Authenticate is true", func() {
				BeforeEach(func() {
					deployadactyl.Config.Environments[environment] = config.Environment{Authenticate: true}
				})

				Context("username and password are provided", func() {
					It("accepts the request with a 200 status", func() {
						deployer.On("Deploy", deploymentInfo, mock.Anything).Run(writeToOut("push succeeded")).Return(nil).Times(1)

						jsonBuffer := bytes.NewBufferString(fmt.Sprintf(`{
							"artifact_url": "%s"
							}`,
							artifactUrl,
						))

						req, err := http.NewRequest("POST", apiUrl, jsonBuffer)
						Expect(err).ToNot(HaveOccurred())

						req.SetBasicAuth(username, password)
						r.ServeHTTP(resp, req)

						Expect(resp.Code).To(Equal(200))
					})
				})

				Context("when username and password are not provided", func() {
					It("rejects the request with a 401 status", func() {
						jsonBuffer := bytes.NewBufferString("{}")
						req, err := http.NewRequest("POST", apiUrl, jsonBuffer)
						Expect(err).ToNot(HaveOccurred())

						r.ServeHTTP(resp, req)
						Expect(resp.Code).To(Equal(401))
					})
				})
			})

			Context("Authenticate is false", func() {
				BeforeEach(func() {
					deployadactyl.Config.Environments[environment] = config.Environment{Authenticate: false}
				})

				Context("username and password are provided", func() {
					It("accepts the request with a 200 status", func() {
						deployer.On("Deploy", deploymentInfo, mock.Anything).Run(writeToOut("push succeeded")).Return(nil).Times(1)

						jsonBuffer := bytes.NewBufferString(fmt.Sprintf(`{
							"artifact_url": "%s"
							}`,
							artifactUrl,
						))

						req, err := http.NewRequest("POST", apiUrl, jsonBuffer)
						Expect(err).ToNot(HaveOccurred())

						req.SetBasicAuth(username, password)
						r.ServeHTTP(resp, req)

						Expect(resp.Code).To(Equal(200))
					})
				})

				Context("username and password are not provided", func() {
					It("accepts the request with a 200 status", func() {
						deploymentInfo.Username = defaultUsername
						deploymentInfo.Password = defaultPassword
						deployer.On("Deploy", deploymentInfo, mock.Anything).Run(writeToOut("push succeeded")).Return(nil).Times(1)

						jsonBuffer := bytes.NewBufferString(fmt.Sprintf(`{
							"artifact_url": "%s"
							}`,
							artifactUrl,
						))

						req, err := http.NewRequest("POST", apiUrl, jsonBuffer)
						Expect(err).ToNot(HaveOccurred())

						r.ServeHTTP(resp, req)

						Expect(resp.Code).To(Equal(200))
					})
				})
			})
		})

		Context("when the request has all necessary parameters", func() {
			BeforeEach(func() {
				jsonBuffer := bytes.NewBufferString(fmt.Sprintf(`{
					"artifact_url": "%s"
					}`,
					artifactUrl,
				))

				var err error
				req, err = http.NewRequest("POST", apiUrl, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())
				req.SetBasicAuth(username, password)
			})

			Context("when deployer succeeds", func() {
				BeforeEach(func() {
					deployer.On("Deploy", deploymentInfo, mock.Anything).Run(writeToOut("push succeeded")).Return(nil).Times(1)
				})

				It("returns a 200 status code", func() {
					r.ServeHTTP(resp, req)
					Expect(resp.Code).To(Equal(200))
				})

				It("responds with the output of the push command", func() {
					r.ServeHTTP(resp, req)
					Expect(resp.Body.String()).To(Equal("push succeeded"))
				})
			})

			Context("when custom manifest information is given in the request body", func() {
				It("properly decodes base64 encoding of that manifest information", func() {
					deploymentInfo.Manifest = "manifest-" + randomizer.StringRunes(10)
					base64Manifest := base64.StdEncoding.EncodeToString([]byte(deploymentInfo.Manifest))

					jsonBuffer := bytes.NewBufferString(fmt.Sprintf(`{
							"artifact_url": "%s",
							"manifest": "%s"
							}`,
						artifactUrl,
						base64Manifest,
					))

					var err error
					req, err = http.NewRequest("POST", apiUrl, jsonBuffer)
					Expect(err).ToNot(HaveOccurred())

					req.SetBasicAuth(username, password)
					deployer.On("Deploy", deploymentInfo, mock.Anything).Run(writeToOut("successful push")).Return(nil).Times(1)

					r.ServeHTTP(resp, req)
					Expect(resp.Code).To(Equal(200))
				})

				It("returns an error if manifest information is not base64 encoded", func() {
					deploymentInfo.Manifest = "manifest-" + randomizer.StringRunes(10)

					jsonBuffer := bytes.NewBufferString(fmt.Sprintf(`{
							"artifact_url": "%s",
							"manifest": "%s"
							}`,
						artifactUrl,
						deploymentInfo.Manifest,
					))

					var err error
					req, err = http.NewRequest("POST", apiUrl, jsonBuffer)
					Expect(err).ToNot(HaveOccurred())

					req.SetBasicAuth(username, password)

					r.ServeHTTP(resp, req)
					Expect(resp.Code).To(Equal(500))
				})
			})

			Context("when deployer fails", func() {
				BeforeEach(func() {
					deployer.On("Deploy", deploymentInfo, mock.Anything).Run(writeToOut("some awesome CF error\n")).Return(errors.New("bork")).Times(1)
				})

				It("returns a 500 status code", func() {
					r.ServeHTTP(resp, req)
					Expect(resp.Code).To(Equal(500))
				})

				It("responds with the output of the push command", func() {
					r.ServeHTTP(resp, req)
					Expect(resp.Body.String()).To(Equal("some awesome CF error\n"))
				})
			})
		})
	})
})

func writeToOut(str string) func(args mock.Arguments) {
	return func(args mock.Arguments) {
		fmt.Fprint(args.Get(1).(io.Writer), str)
	}
}
