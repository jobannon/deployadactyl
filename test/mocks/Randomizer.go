package mocks

type Randomizer struct {
	RandomizeCall struct {
		Received struct {
			Length int
		}
		Returns struct {
			Runes string
		}
	}
}

func (r *Randomizer) StringRunes(length int) string {
	r.RandomizeCall.Received.Length = length

	return r.RandomizeCall.Returns.Runes
}
