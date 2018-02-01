package error_finder_test

import (
	. "github.com/compozed/deployadactyl/controller/deployer/error_finder"

	"github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"reflect"
)

var _ = Describe("ErrorFinder", func() {
	It("returns a TrustStoreError when the response to the user shows a trust store error", func() {
		response := "Creating TrustStore with container certificates\nFAILED"
		errorFinder := ErrorFinder{}
		err := errorFinder.FindError(response)

		Expect(reflect.TypeOf(err).String()).To(Equal("error_finder.TrustStoreError"))
	})

	It("returns a nil when it cannot detect the error", func() {
		errorFinder := ErrorFinder{}
		err := errorFinder.FindError("Good luck catching this error")

		Expect(err).To(BeNil())
	})

	It("returns no errors when no matchers are configured", func() {
		errorFinder := ErrorFinder{}
		errors := errorFinder.FindErrors("This is some text that doesn't affect the test")

		Expect(len(errors)).To(BeZero())
	})

	It("returns multiple errors when matchers are configured", func() {
		matchers := make([]interfaces.ErrorMatcher, 0, 0)

		matcher := &mocks.ErrorMatcherMock{}
		matcher.MatchCall.Returns = CreateDeploymentError("a test error", []string{"error 1", "error 2", "error 3"})
		matchers = append(matchers, matcher)

		matcher = &mocks.ErrorMatcherMock{}
		matcher.MatchCall.Returns = CreateDeploymentError("another test error", []string{"error 4", "error 5", "error 6"})
		matchers = append(matchers, matcher)

		errorFinder := ErrorFinder{Matchers: matchers}
		errors := errorFinder.FindErrors("This is some text that doesn't affect the test")

		Expect(len(errors)).To(Equal(2))
		Expect(errors[0].Error()).To(Equal("a test error"))
		Expect(errors[0].Details()[0]).To(Equal("error 1"))
		Expect(errors[1].Error()).To(Equal("another test error"))
		Expect(errors[1].Details()[2]).To(Equal("error 6"))
	})

})

var _ = Describe("WriteErrors", func() {
	//It("writes multiple errors to writer", func() {
	//	errors := []string{"test1", "test2", "test3"}
	//
	//	response := ErrorFinder{}.WriteErrors(errors)
	//
	//})
})
