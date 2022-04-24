package source

import (
	"github.com/nvkalinin/business-calendar/store"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOverride_GetYear(t *testing.T) {
	ov := &Override{
		Path: "testdata/override.yml",
	}

	months, err := ov.GetYear(2022)
	expMonths := store.Months{
		11: store.Days{
			4: {Type: store.Normal},
			7: {Type: store.Holiday, Desc: "День Великой Октябрьской социалистической революции"},
		},
	}
	assert.NoError(t, err)
	assert.Equal(t, expMonths, months)

	months, err = ov.GetYear(2023)
	assert.NoError(t, err)
	assert.Len(t, months, 0)
}
