package bluegreen_test

import (
	"errors"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Actor", func() {

	FContext("When actors are given commands", func() {
		It("returns error on failure", func() {
			foundationURL := "www.something.com"
			action := &mocks.Action{}
			action.ExecuteCall.Returns.Error = errors.New("error")
			a := bluegreen.NewActor(action, foundationURL)
			a.Commands <- func(action interfaces.Action, foundationURL string) error {
				return action.Execute()
			}
			Expect((<-a.Errs).Error()).To(Equal("error"))
		})
		It("doesn't return an error on success", func() {
			foundationURL := "www.something.com"
			action := &mocks.Action{}
			a := bluegreen.NewActor(action, foundationURL)
			a.Commands <- func(action interfaces.Action, foundationURL string) error {
				return action.Execute()
			}
			Expect(<-a.Errs).ToNot(HaveOccurred())
		})
	})
})
