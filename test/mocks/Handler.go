package mocks

import S "github.com/compozed/deployadactyl/structs"

type Handler struct {
	OnEventCall struct {
		Received struct {
			Event S.Event
		}
		Returns struct {
			Error error
		}
	}
}

func (h *Handler) OnEvent(event S.Event) error {
	h.OnEventCall.Received.Event = event

	return h.OnEventCall.Returns.Error
}
