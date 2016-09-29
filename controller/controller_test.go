package controller_test

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

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

		apiURL      string
		appName     string
		environment string
		org         string
		space       string
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
		space = "space-" + randomizer.StringRunes(10)

		router.POST("/v1/apps/:environment/:org/:space/:appName", controller.Deploy)
	})

	Describe("Deploy handler", func() {
		Context("when deployer succeeds", func() {
			It("deploys and returns http.StatusOK", func() {
				apiURL = fmt.Sprintf("/v1/apps/%s/%s/%s/%s", environment, org, space, appName)

				req, err := http.NewRequest("POST", apiURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())

				deployer.DeployCall.Returns.Error = nil
				deployer.DeployCall.Returns.StatusCode = http.StatusOK
				deployer.DeployCall.Write.Output = "deploy success"

				router.ServeHTTP(resp, req)

				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(resp.Body).To(ContainSubstring("deploy success"))

				Expect(deployer.DeployCall.Received.Environment).To(Equal(environment))
				Expect(deployer.DeployCall.Received.Org).To(Equal(org))
				Expect(deployer.DeployCall.Received.Space).To(Equal(space))
				Expect(deployer.DeployCall.Received.AppName).To(Equal(appName))
			})
		})

		Context("when deployer fails", func() {
			It("doesn't deploy and gives http.StatusInternalServerError", func() {
				apiURL = fmt.Sprintf("/v1/apps/%s/%s/%s/%s", environment, org, space, appName)

				req, err := http.NewRequest("POST", apiURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())

				deployer.DeployCall.Returns.Error = errors.New("bork")
				deployer.DeployCall.Returns.StatusCode = http.StatusInternalServerError

				router.ServeHTTP(resp, req)

				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body).To(ContainSubstring("bork"))
			})
		})
	})
})
