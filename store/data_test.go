package store

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestYear_Copy(t *testing.T) {
	yOrig := Year{
		2022: Month{
			1: {Working: true, Type: Holiday},
		},
	}

	yCopy := yOrig.Copy()
	yCopy[2022][1] = Day{Working: false, Type: Normal}

	assert.NotEqual(t, yCopy, yOrig)
}
