package service_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	ENDPOINT        = "/v1/apps/:environment/:org/:space/:appName"
	CONFIGPATH      = "./test_config.yml"
	ENVIRONMENTNAME = "test"
	TESTCONFIG      = `---
environments:
- name: Test
  domain: test.example.com
  skip_ssl: true
  foundations:
  - api1.example.com
  - api2.example.com
  - api3.example.com
  - api4.example.com
`
)

var _ = Describe("Service", func() {

	var (
		deployadactylServer *httptest.Server
		artifactServer      *httptest.Server
		org                 = randomizer.StringRunes(10)
		space               = randomizer.StringRunes(10)
		appName             = randomizer.StringRunes(10)
	)

	BeforeEach(func() {
		os.Setenv("CF_USERNAME", randomizer.StringRunes(10))
		os.Setenv("CF_PASSWORD", randomizer.StringRunes(10))

		Expect(ioutil.WriteFile(CONFIGPATH, []byte(TESTCONFIG), 0644)).To(Succeed())

		creator, err := mocks.NewCreator("debug", CONFIGPATH)
		Expect(err).ToNot(HaveOccurred())

		deployadactylHandler := creator.CreateControllerHandler()

		deployadactylServer = httptest.NewServer(deployadactylHandler)

		artifactServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "fixtures/artifact-with-manifest.jar")
		}))
	})

	AfterEach(func() {
		Expect(os.Remove(CONFIGPATH)).To(Succeed())
		deployadactylServer.Close()
		artifactServer.Close()
	})

	Context("mocking the courier and the prechecker", func() {
		Context("receiving an artifact url", func() {
			It("can deploy an application without the internet", func() {
				j, err := json.Marshal(gin.H{
					"artifact_url": artifactServer.URL,
				})
				Expect(err).ToNot(HaveOccurred())
				jsonBuffer := bytes.NewBuffer(j)

				requestURL := fmt.Sprintf("%s/v1/apps/%s/%s/%s/%s", deployadactylServer.URL, ENVIRONMENTNAME, org, space, appName)
				req, err := http.NewRequest("POST", requestURL, jsonBuffer)
				Expect(err).ToNot(HaveOccurred())

				req.Header.Add("Content-Type", "application/json")

				client := &http.Client{}
				resp, err := client.Do(req)
				Expect(err).ToNot(HaveOccurred())

				responseBody, err := ioutil.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(http.StatusOK), string(responseBody))

				fmt.Fprintf(GinkgoWriter, "\nUser Output:\n%s\n%s\n%s", strings.Repeat("-", 60), string(responseBody), strings.Repeat("-", 60))
			})
		})

		Context("receiving an artifact in the request body", func() {
			It("can deploy an application without the internet", func() {
				body, err := os.Open("fixtures/artifact-with-manifest.jar")
				Expect(err).ToNot(HaveOccurred())

				requestURL := fmt.Sprintf("%s/v1/apps/%s/%s/%s/%s", deployadactylServer.URL, ENVIRONMENTNAME, org, space, appName)
				req, err := http.NewRequest("POST", requestURL, body)
				Expect(err).ToNot(HaveOccurred())

				req.Header.Add("Content-Type", "application/zip")

				client := &http.Client{}
				resp, err := client.Do(req)
				Expect(err).ToNot(HaveOccurred())

				responseBody, err := ioutil.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(http.StatusOK), string(responseBody))
			})
		})
	})
})
