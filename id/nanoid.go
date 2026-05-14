// Package id provides URL-safe unique ID generation backed by nanoid.
//
// The default generator produces 16-character IDs from an alphanumeric alphabet
// (0-9, A-Z, a-z), giving ~95 bits of entropy — suitable for record IDs, request
// trace IDs, and other unique identifiers.
//
//	gen := id.NewGenerator()
//	id  := gen.GenerateID() // e.g. "aB3kX7mNpQ2rSt4v"
//
// Use [NewGeneratorWithLength] or [NewGeneratorWithAlphabet] when you need a
// different size or character set.
package id

import (
	nanoid "github.com/matoous/go-nanoid/v2"
)

const (
	defaultAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	defaultLength   = 16
)

type Generator interface {
	GenerateID() string
}

type nanoIDGenerator struct {
	alphabet string
	length   int
}

func NewGenerator() Generator {
	return &nanoIDGenerator{alphabet: defaultAlphabet, length: defaultLength}
}

func NewGeneratorWithAlphabet(alphabet string, length int) Generator {
	if alphabet == "" || length <= 0 {
		panic("id: alphabet must be non-empty and length must be positive")
	}
	return &nanoIDGenerator{alphabet: alphabet, length: length}
}

func NewGeneratorWithLength(length int) Generator {
	if length <= 0 {
		panic("id: length must be positive")
	}
	return &nanoIDGenerator{alphabet: defaultAlphabet, length: length}
}

func (g *nanoIDGenerator) GenerateID() string {
	id, err := nanoid.Generate(g.alphabet, g.length)
	if err != nil {
		panic("id: failed to generate id: " + err.Error())
	}
	return id
}
