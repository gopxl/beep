package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClamp(t *testing.T) {
	assert.Equal(t, 0, Clamp(-5, 0, 1))
	assert.Equal(t, 1, Clamp(5, 0, 1))
	assert.Equal(t, 0.5, Clamp(0.5, 0, 1))
}
