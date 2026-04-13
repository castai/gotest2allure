package main

import (
	"strings"

	"github.com/google/uuid"
)

const RFC3339Milli = "2006-01-02T15:04:05.999Z07:00"

func Map[T any, R any](slice []T, fn func(T) R) []R {
	result := make([]R, len(slice))
	for i, v := range slice {
		result[i] = fn(v)
	}
	return result
}

func hash(s ...string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(strings.Join(s, "/"))).String()
}
