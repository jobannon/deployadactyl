package push_test

import (
	"github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/state/push"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("event binding", func() {
	Describe("DeployStartedEvent", func() {
		Describe("Accept", func() {
			Context("when accept takes a correct event", func() {
				It("should return true", func() {
					binding := push.NewDeployStartEventBinding(nil)

					event := push.DeployStartedEvent{}
					Expect(binding.Accepts(event)).Should(Equal(true))
				})
			})
			Context("when accept takes incorrect event", func() {
				It("should return false", func() {
					binding := push.NewDeployStartEventBinding(nil)

					event := interfaces.Event{}
					Expect(binding.Accepts(event)).Should(Equal(false))
				})
			})
		})
		Describe("Emit", func() {
			Context("when emit takes a correct event", func() {
				It("should invoke handler", func() {
					invoked := false
					handler := func(event push.DeployStartedEvent) error {
						invoked = true
						return nil
					}
					binding := push.NewDeployStartEventBinding(handler)
					event := push.DeployStartedEvent{}
					binding.Emit(event)

					Expect(invoked).Should(Equal(true))
				})
			})
			Context("when emit takes incorrect event", func() {
				It("should return error", func() {
					invoked := false
					handler := func(event push.DeployStartedEvent) error {
						invoked = true
						return nil
					}
					binding := push.NewDeployStartEventBinding(handler)
					event := interfaces.Event{}
					err := binding.Emit(event)

					Expect(invoked).Should(Equal(false))
					Expect(err).ShouldNot(BeNil())
					Expect(err.Error()).Should(Equal("invalid event type"))
				})
			})
		})
	})

	Describe("DeployFinishEvent", func() {
		Describe("Accept", func() {
			Context("when accept takes a correct event", func() {
				It("should return true", func() {
					binding := push.NewDeployFinishedEventBinding(nil)

					event := push.DeployFinishedEvent{}
					Expect(binding.Accepts(event)).Should(Equal(true))
				})
			})
			Context("when accept takes incorrect event", func() {
				It("should return false", func() {
					binding := push.NewDeployFinishedEventBinding(nil)

					event := interfaces.Event{}
					Expect(binding.Accepts(event)).Should(Equal(false))
				})
			})
		})
		Describe("Emit", func() {
			Context("when emit takes a correct event", func() {
				It("should invoke handler", func() {
					invoked := false
					pushFunc := func(event push.DeployFinishedEvent) error {
						invoked = true
						return nil
					}
					binding := push.NewDeployFinishedEventBinding(pushFunc)
					event := push.DeployFinishedEvent{}
					binding.Emit(event)

					Expect(invoked).Should(Equal(true))
				})
			})
			Context("when emit takes incorrect event", func() {
				It("should return error", func() {
					invoked := false
					pushFunc := func(event push.DeployFinishedEvent) error {
						invoked = true
						return nil
					}
					binding := push.NewDeployFinishedEventBinding(pushFunc)
					event := interfaces.Event{}
					err := binding.Emit(event)

					Expect(invoked).Should(Equal(false))
					Expect(err).ShouldNot(BeNil())
					Expect(err.Error()).Should(Equal("invalid event type"))
				})
			})
		})
	})
})
