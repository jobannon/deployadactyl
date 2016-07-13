package eventmanager_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/op/go-logging"

	. "github.com/compozed/deployadactyl/deployer/eventmanager"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/compozed/deployadactyl/test/mocks"
)

var _ = Describe("Events", func() {
	var (
		eventType string
		eventData string
		logBuffer *gbytes.Buffer
		log       *logging.Logger
	)

	BeforeEach(func() {
		eventType = "eventType-" + randomizer.StringRunes(10)
		eventData = "eventData-" + randomizer.StringRunes(10)
		logBuffer = gbytes.NewBuffer()
		log = logger.DefaultLogger(logBuffer, logging.DEBUG, "eventmanager_test")
	})

	Context("When an event handler is registered", func() {
		It("should be successful", func() {
			eventManager := NewEventManager(log)
			eventHandler := &mocks.Handler{}

			Expect(eventManager.AddHandler(eventHandler, eventType)).To(Succeed())
		})

		It("should fail if a nil value is passed in as an argument", func() {
			eventManager := NewEventManager(log)

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
			eventManager := NewEventManager(log)

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
			eventManager := NewEventManager(log)

			eventManager.AddHandler(eventHandler, eventType)
			Expect(eventManager.Emit(event)).ToNot(Succeed())
			Expect(eventHandler.AssertExpectations(GinkgoT())).To(BeTrue())
		})

		It("should log that the event is emitted", func() {
			event := S.Event{Type: eventType, Data: eventData}
			eventHandler := &mocks.Handler{}
			eventHandler.On("OnEvent", event).Return(nil).Times(1)
			eventManager := NewEventManager(log)
			eventManager.AddHandler(eventHandler, eventType)
			Expect(eventManager.Emit(event)).To(Succeed())
			Eventually(logBuffer).Should(gbytes.Say("An event %s has been emitted", eventType))
		})
	})

	Context("when there are handlers registered for two different types of events", func() {
		It("only emits to the specified event", func() {
			event := S.Event{Type: eventType, Data: eventData}
			eventManager := NewEventManager(log)
			eventHandlerOne := &mocks.Handler{}
			eventHandlerTwo := &mocks.Handler{}
			eventHandlerOne.On("OnEvent", event).Return(nil).Times(1)

			eventManager.AddHandler(eventHandlerOne, eventType)
			eventManager.AddHandler(eventHandlerTwo, "anotherEventType-"+randomizer.StringRunes(10))
			Expect(eventManager.Emit(event)).To(Succeed())
		})
	})
})
