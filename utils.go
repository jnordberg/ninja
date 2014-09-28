package main

import (
	"math/rand"
)

var CodeChars = []rune("abcdefghjkmnpqrstuvwxyz")

func GenerateCode(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = CodeChars[rand.Intn(len(CodeChars))]
	}
	return string(b)
}
