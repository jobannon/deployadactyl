package mocks

import S "github.com/compozed/deployadactyl/structs"

// Handler handmade mock for tests.
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

// OnEvent mock method.
func (h *Handler) OnEvent(event S.Event) error {
	h.OnEventCall.Received.Event = event

	return h.OnEventCall.Returns.Error
}
