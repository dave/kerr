package kerrsource

import (
	"testing"

	"kego.io/kerr/assert"
)

func TestIsId(t *testing.T) {
	assert.True(t, IsId("LWMRGATIAT"))
	assert.False(t, IsId("LWMRGATIA"))
	assert.False(t, IsId("LWMRGATIAt"))
	assert.False(t, IsId("LWMRGAT9AT"))
}
