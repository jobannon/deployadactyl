package eventmanager_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/compozed/conveyor/test"
	. "github.com/compozed/deployadactyl/deployer/eventmanager"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/compozed/deployadactyl/test/mocks"
)

var _ = Describe("Events", func() {
	var (
		eventType string
		eventData string
	)

	BeforeEach(func() {
		eventType = "eventType-" + test.RandStringRunes(10)
		eventData = "eventData-" + test.RandStringRunes(10)
	})

	Context("When an event handler is registered", func() {
		It("should be successful", func() {
			eventManager := NewEventManager()
			eventHandler := &mocks.Handler{}

			Expect(eventManager.AddHandler(eventHandler, eventType)).To(Succeed())
		})

		It("should fail if a nil value is passed in as an argument", func() {
			eventManager := NewEventManager()

			Expect(eventManager.AddHandler(nil, eventType)).ToNot(Succeed())
		})
	})

	Context("When an event is emitted", func() {
		It("should call all event handlers", func() {
			event := S.Event{Type: eventType, Data: eventData}
			eventHandlerOne := &mocks.Handler{}
			eventHandlerOne.On("OnEvent", event).Return(nil).Times(1)
			eventHandlerTwo := &mocks.Handler{}
			eventHandlerTwo.On("OnEvent", event).Return(nil).Times(1)
			eventManager := NewEventManager()

			eventManager.AddHandler(eventHandlerOne, eventType)
			eventManager.AddHandler(eventHandlerTwo, eventType)
			Expect(eventManager.Emit(event)).To(Succeed())
			Expect(eventHandlerOne.AssertExpectations(GinkgoT())).To(BeTrue())
			Expect(eventHandlerTwo.AssertExpectations(GinkgoT())).To(BeTrue())
		})

		It("should return an error if the handler returns an error", func() {
			event := S.Event{Type: eventType, Data: eventData}
			eventHandler := &mocks.Handler{}
			eventHandler.On("OnEvent", event).Return(errors.New("bork")).Times(1)
			eventManager := NewEventManager()

			eventManager.AddHandler(eventHandler, eventType)
			Expect(eventManager.Emit(event)).ToNot(Succeed())
			Expect(eventHandler.AssertExpectations(GinkgoT())).To(BeTrue())
		})
	})

	Context("when there are handlers registered for two different types of events", func() {
		It("only emits to the specified event", func() {
			event := S.Event{Type: eventType, Data: eventData}
			eventManager := NewEventManager()
			eventHandlerOne := &mocks.Handler{}
			eventHandlerTwo := &mocks.Handler{}
			eventHandlerOne.On("OnEvent", event).Return(nil).Times(1)

			eventManager.AddHandler(eventHandlerOne, eventType)
			eventManager.AddHandler(eventHandlerTwo, "anotherEventType-"+test.RandStringRunes(10))
			Expect(eventManager.Emit(event)).To(Succeed())
		})
	})
})
