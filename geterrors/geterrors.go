package geterrors

import (
	"fmt"
	"strings"
)

func WrapFunc(get func(string) string) ErrGetter {
	return ErrGetter{get: get}
}

type ErrGetter struct {
	get         func(string) string
	missingKeys []string
}

func (g *ErrGetter) Get(key string) string {
	val := g.get(key)
	if len(val) == 0 {
		g.missingKeys = append(g.missingKeys, key)
	}
	return val
}

func (g *ErrGetter) Err(message string) error {
	if len(g.missingKeys) == 0 {
		return nil
	}
	return fmt.Errorf(
		"%s: %s",
		message, strings.Join(g.missingKeys, ", "),
	)
}
