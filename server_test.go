package main_test

import (
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var goodConfig = []byte(`---
environments:
  - name: test
    domain: examples.are.cool.com
    foundations:
    - https://example.endpoint1.cf.com
    - https://example.endpoint2.cf.com
    allow_page: false
    authenticate: false
    skip_ssl: true

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

	Describe("log level", func() {
		Context("when a log level is not specified", func() {
			It("uses the default log level ", func() {
				level := os.Getenv("DEPLOYADACTYL_LOGLEVEL")

				os.Unsetenv("DEPLOYADACTYL_LOGLEVEL")
				Expect(err).ToNot(HaveOccurred())

				session, err = gexec.Start(exec.Command(pathToCLI), GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())

				Eventually(session.Out).Should(Say("log level"))
				Eventually(session.Out).Should(Say("DEBUG"))

				os.Setenv("DEPLOYADACTYL_LOGLEVEL", level)
			})
		})

		Context("when log level is invalid", func() {
			It("throws an error", func() {
				level := os.Getenv("DEPLOYADACTYL_LOGLEVEL")

				Expect(os.Setenv("DEPLOYADACTYL_LOGLEVEL", "tanystropheus")).To(Succeed())

				session, err = gexec.Start(exec.Command(pathToCLI), GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())

				Eventually(session.Err).Should(Say("invalid log level"))

				os.Setenv("DEPLOYADACTYL_LOGLEVEL", level)
			})
		})
	})

	Context("when an invalid config path is specified", func() {
		It("throws an error", func() {
			session, err = gexec.Start(exec.Command(pathToCLI, "-config", "./gorgosaurus.yml"), GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())

			Eventually(session.Out).Should(Say("no such file or directory"))
		})
	})
})
