package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClamp(t *testing.T) {
	assert.Equal(t, 0.0, ClampFloat64(-5, 0, 1))
	assert.Equal(t, 1.0, ClampFloat64(5, 0, 1))
	assert.Equal(t, 0.5, ClampFloat64(0.5, 0, 1))
}
