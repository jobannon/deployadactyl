package main_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var goodConfig = []byte(`---
environments:
  - name: sandbox
    service_now_endpoint: https://allstateuat.service-now.com/u_platform_integration.do?SOAP
    domain: platformtest.allstate.com
    allow_page: false
    authenticate: false
    skip_ssl: true
    foundations:
    - https://api.cf.sandbox-mpn.ro98.allstate.com
    - https://api.cf.sandbox-mpn.ro99.allstate.com
`)

var badTestConfig = []byte(`---
environments:
  - name: sandbox
`)

var _ = Describe("Server", func() {

	var (
		session *gexec.Session
		err     error
	)

	AfterEach(func() {
		session.Terminate()
	})

	It("uses the default log level when a log level is not specified", func() {
		configLocation := fmt.Sprintf("%s/config.yml", path.Dir(pathToCLI))

		Expect(ioutil.WriteFile(configLocation, goodConfig, 0777)).To(Succeed())

		level := os.Getenv("DEPLOYADACTYL_LOGLEVEL")

		os.Unsetenv("DEPLOYADACTYL_LOGLEVEL")
		Expect(err).ToNot(HaveOccurred())

		session, err = gexec.Start(exec.Command(pathToCLI, "-config", configLocation), GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		Eventually(session.Out).Should(gbytes.Say("log level"))
		Eventually(session.Out).Should(gbytes.Say("DEBUG"))

		os.Setenv("DEPLOYADACTYL_LOGLEVEL", level)
	})

	It("throws an error when log level is invalid", func() {
		level := os.Getenv("DEPLOYADACTYL_LOGLEVEL")
		err = os.Setenv("DEPLOYADACTYL_LOGLEVEL", "tanystropheus")
		Expect(err).ToNot(HaveOccurred())

		session, err = gexec.Start(exec.Command(pathToCLI), GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		Eventually(session.Err).Should(gbytes.Say("invalid log level"))

		os.Setenv("DEPLOYADACTYL_LOGLEVEL", level)
	})

	It("throws an error when an invalid config path is specified", func() {
		session, err = gexec.Start(exec.Command(pathToCLI, "-config", "./gorgosaurus.yml"), GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		Eventually(session.Out).Should(gbytes.Say("no such file or directory"))
	})
})
