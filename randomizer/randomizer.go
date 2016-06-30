// Package randomizer is used for generating random strings.
package randomizer

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// StringRunes generates a random string of runes of a specified length.
func StringRunes(n int) string {
	return generateRunes(n)
}

type Randomizer struct{}

// StringRunes generates a random string of runes of a specified length from a Randomizer struct.
func (r Randomizer) StringRunes(n int) string {
	return generateRunes(n)
}

func generateRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
