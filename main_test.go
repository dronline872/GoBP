package main

import (
	"testing"

	"github.com/go-playground/assert"
)

func TestPath(t *testing.T) {
	file := fileInfo{
		path: "/test/test",
	}

	assert.Equal(t, "/test/test", file.Path())
}
