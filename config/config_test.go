package config_test

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/compozed/conveyor/test"
	. "github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/test/mocks"
)

const (
	configPath = "./test_config.yml"
	testConfig = `---
environments:
- name: Test
  domain: test.example.com
  foundations:
  - api1.example.com
  - api2.example.com
  skip_ssl: true
- name: Prod
  domain: example.com
  foundations:
  - api3.example.com
  - api4.example.com
  skip_ssl: false
`
)

var _ = Describe("Config", func() {
	var (
		env        *mocks.Env
		envMap     map[string]Environment
		cfUsername string
		cfPassword string
	)

	BeforeEach(func() {
		cfUsername = "cfUsername-" + test.RandStringRunes(10)
		cfPassword = "cfPassword-" + test.RandStringRunes(10)

		env = &mocks.Env{}

		envMap = map[string]Environment{
			"test": Environment{
				Name:        "Test",
				Foundations: []string{"api1.example.com", "api2.example.com"},
				Domain:      "test.example.com",
				SkipSSL:     true,
			},
			"prod": Environment{
				Name:        "Prod",
				Foundations: []string{"api3.example.com", "api4.example.com"},
				Domain:      "example.com",
				SkipSSL:     false,
			},
		}

		Expect(ioutil.WriteFile(configPath, []byte(testConfig), 0644)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(configPath)).To(Succeed())
	})

	Context("when all environment variables are present", func() {
		JustBeforeEach(func() {
			env.On("Get", "CF_USERNAME").Return(cfUsername)
			env.On("Get", "CF_PASSWORD").Return(cfPassword)
			env.On("Get", "PORT").Return("")
		})

		It("returns a valid Config", func() {
			config, err := New(env.Get, configPath)
			Expect(err).ToNot(HaveOccurred())

			Expect(config.Username).To(Equal(cfUsername))
			Expect(config.Password).To(Equal(cfPassword))
			Expect(config.Environments).To(Equal(envMap))
			Expect(config.Port).To(Equal(8080))
		})

		Context("when PORT is in the environment", func() {
			BeforeEach(func() {
				env.On("Get", "PORT").Return("42")
			})

			It("uses the value as the Port", func() {
				config, err := New(env.Get, configPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(config.Port).To(Equal(42))
			})
		})
	})

	Context("when an environment variable is missing", func() {
		BeforeEach(func() {
			env.On("Get", "CF_USERNAME").Return("")
			env.On("Get", "CF_PASSWORD").Return(cfPassword)
		})

		It("returns an error", func() {
			_, err := New(env.Get, configPath)
			Expect(err).To(MatchError("missing environment variables: CF_USERNAME"))
		})
	})
})
