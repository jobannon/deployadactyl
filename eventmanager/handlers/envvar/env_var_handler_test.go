package handlers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/op/go-logging"
	"github.com/spf13/afero"

	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	S "github.com/compozed/deployadactyl/structs"
)

var _ = Describe("Env_Var_Handler", func() {
	var (
		eventHandler Envvarhandler
		logBuffer    *gbytes.Buffer
		log          I.Logger
		event        S.Event
		filesystem   = &afero.Afero{Fs: afero.NewMemMapFs()}
	)

	BeforeEach(func() {
		logBuffer = gbytes.NewBuffer()
		log = logger.DefaultLogger(logBuffer, logging.DEBUG, "evn_var_handler_test")
		event = S.Event{Type: "test-event", Data: S.DeployEventData{}}
		eventHandler = Envvarhandler{Logger: log, FileSystem: filesystem}
	})

	Context("when an envvarhandler is called with event without deploy info", func() {
		It("it should be succeed", func() {

			Expect(eventHandler.OnEvent(event)).To(Succeed())
		})

	})

	Context("when an envvarhandler is called with event without env variables", func() {
		It("it should be succeed", func() {

			event.Data = S.DeployEventData{DeploymentInfo: &S.DeploymentInfo{}}

			Expect(eventHandler.OnEvent(event)).To(Succeed())
		})

	})

	Context("when an envvarhandler is called with event with env variables", func() {
		It("it should be succeed", func() {

			path := "/tmp"
			eventHandler.FileSystem.MkdirAll(path, 0755)

			envvars := make(map[string]string)
			envvars["one"] = "one"
			envvars["two"] = "two"

			info := S.DeploymentInfo{
				AppName:              "testApp",
				AppPath:              path,
				EnvironmentVariables: envvars,
			}

			event.Data = S.DeployEventData{DeploymentInfo: &info}

			//Process the event
			Expect(eventHandler.OnEvent(event)).To(Succeed())

			//Verify manifest was written and matches
			manifest, err := ReadManifest(path+"/manifest.yml", eventHandler.Logger, eventHandler.FileSystem)

			Expect(manifest).NotTo(BeNil())
			Expect(err).To(BeNil())
			Expect(manifest.Content.Applications[0].Name).To(Equal("testApp"))
			Expect(len(manifest.Content.Applications[0].Env)).To(Equal(2))
		})
	})

	Context("when an envvarhandler is called with bogus manifest in deploy info", func() {
		It("it should be fail", func() {

			content := `bork`

			envvars := make(map[string]string)
			envvars["one"] = "one"
			envvars["two"] = "two"

			info := S.DeploymentInfo{
				AppName:              "testApp",
				AppPath:              "/tmp",
				Manifest:             content,
				EnvironmentVariables: envvars,
			}

			event.Data = S.DeployEventData{DeploymentInfo: &info}

			err := eventHandler.OnEvent(event)

			Expect(err).ToNot(BeNil())
		})

	})

})
