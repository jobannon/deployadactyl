package error_finder

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("RegExErrorMatcher", func() {
	It("should match no errors", func() {
		factory := ErrorMatcherFactory{}
		errorMatcher, _ := factory.CreateErrorMatcher("this shouldn't bring back errors", ".{1,10}regex stuff.{1,20}")
		err := errorMatcher.Match([]byte("this does not contain the regex"))
		Expect(err).To(BeNil())
	})

	It("should match one error", func() {
		factory := ErrorMatcherFactory{}
		errorMatcher, _ := factory.CreateErrorMatcher("this should bring back one error", ".{1,10}ab.{1,20}")
		err := errorMatcher.Match([]byte("xxxxxabxxxxxxx"))
		Expect(len(err.Details())).To(Equal(1))
		Expect(err.Error()).To(Equal("this should bring back one error"))
		Expect(err.Details()[0]).To(Equal("xxxxxabxxxxxxx"))
	})

	It("should match multiple errors", func() {
		factory := ErrorMatcherFactory{}
		errorMatcher, _ := factory.CreateErrorMatcher("this should bring back multiple errors", "ab[^ab]{1,5}")
		err := errorMatcher.Match([]byte("xxxxxabxxxabxxxxxxxxxxxxxxxxxxabx"))
		Expect(len(err.Details())).To(Equal(3))
		Expect(err.Error()).To(Equal("this should bring back multiple errors"))
		Expect(err.Details()[0]).To(Equal("abxxx"))
		Expect(err.Details()[1]).To(Equal("abxxxxx"))
		Expect(err.Details()[2]).To(Equal("abx"))
	})
})
