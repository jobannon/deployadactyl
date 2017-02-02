package eventmanager_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/op/go-logging"

	. "github.com/compozed/deployadactyl/eventmanager"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
)

var _ = Describe("Events", func() {
	var (
		eventType       string
		eventData       string
		eventHandler    *mocks.Handler
		eventHandlerOne *mocks.Handler
		eventHandlerTwo *mocks.Handler
		eventManager    *EventManager
		logBuffer       *gbytes.Buffer
		log             I.Logger
	)

	BeforeEach(func() {
		eventType = "eventType-" + randomizer.StringRunes(10)
		eventData = "eventData-" + randomizer.StringRunes(10)

		eventHandler = &mocks.Handler{}
		eventHandlerOne = &mocks.Handler{}
		eventHandlerTwo = &mocks.Handler{}

		eventManager = NewEventManager(log)

		logBuffer = gbytes.NewBuffer()

		log = logger.DefaultLogger(logBuffer, logging.DEBUG, "eventmanager_test")
	})

	Context("when an event handler is registered", func() {
		It("should be successful", func() {
			eventManager := NewEventManager(log)

			Expect(eventManager.AddHandler(eventHandler, eventType)).To(Succeed())
		})

		It("should fail if a nil value is passed in as an argument", func() {
			eventManager := NewEventManager(log)

			err := eventManager.AddHandler(nil, eventType)

			Expect(err).To(MatchError(InvalidArgumentError{}))
		})
	})

	Context("when an event is emitted", func() {
		It("should call all event handlers", func() {
			eventHandlerOne.OnEventCall.Returns.Error = nil
			eventHandlerTwo.OnEventCall.Returns.Error = nil

			event := S.Event{Type: eventType, Data: eventData}

			eventManager.AddHandler(eventHandlerOne, eventType)
			eventManager.AddHandler(eventHandlerTwo, eventType)

			Expect(eventManager.Emit(event)).To(Succeed())

			Expect(eventHandlerOne.OnEventCall.Received.Event).To(Equal(event))
			Expect(eventHandlerTwo.OnEventCall.Received.Event).To(Equal(event))
		})

		It("should return an error if the handler returns an error", func() {
			eventHandler.OnEventCall.Returns.Error = errors.New("on event error")

			event := S.Event{Type: eventType, Data: eventData}

			eventManager.AddHandler(eventHandler, eventType)

			Expect(eventManager.Emit(event)).To(MatchError("on event error"))
			Expect(eventHandler.OnEventCall.Received.Event).To(Equal(event))
		})

		It("should log that the event is emitted", func() {
			eventHandler.OnEventCall.Returns.Error = nil

			event := S.Event{Type: eventType, Data: eventData}

			eventManager.AddHandler(eventHandler, eventType)

			Expect(eventManager.Emit(event)).To(Succeed())

			Expect(eventHandler.OnEventCall.Received.Event).To(Equal(event))
			Eventually(logBuffer).Should(gbytes.Say("a %s event has been emitted", eventType))
		})
	})

	Context("when there are handlers registered for two different types of events", func() {
		It("only emits to the specified event", func() {
			eventHandlerOne.OnEventCall.Returns.Error = nil
			eventHandlerTwo.OnEventCall.Returns.Error = nil

			event := S.Event{Type: eventType, Data: eventData}

			eventManager.AddHandler(eventHandlerOne, eventType)
			eventManager.AddHandler(eventHandlerTwo, "anotherEventType-"+randomizer.StringRunes(10))

			Expect(eventManager.Emit(event)).To(Succeed())

			Expect(eventHandlerOne.OnEventCall.Received.Event).To(Equal(event))
			Expect(eventHandlerTwo.OnEventCall.Received.Event).ToNot(Equal(event))
		})
	})
})
