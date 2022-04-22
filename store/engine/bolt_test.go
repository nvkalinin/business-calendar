package engine

import (
	"os"
	"testing"
	"time"

	"github.com/nvkalinin/business-calendar/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sample2022 = store.Months{
	1: store.Days{
		1: store.Day{WeekDay: store.Saturday, Working: false, Type: store.Holiday},
		2: store.Day{WeekDay: store.Sunday, Working: false, Type: store.Holiday},
	},
	2: store.Days{
		1: store.Day{WeekDay: store.Tuesday, Working: true, Type: store.Normal},
	},
}

func TestBolt(t *testing.T) {
	b, _ := makeBolt(t)
	defer b.Close()

	err := b.PutYear(2022, sample2022)
	require.NoError(t, err)

	y, ok := b.FindYear(2022)
	assert.True(t, ok)
	assert.Equal(t, sample2022, y)

	m, ok := b.FindMonth(2022, time.February)
	assert.True(t, ok)
	assert.Equal(t, sample2022[time.February], m)

	d, ok := b.FindDay(2022, time.January, 2)
	assert.True(t, ok)
	assert.Equal(t, sample2022[time.January][2], *d)
}

func TestBolt_backup(t *testing.T) {
	b, dir := makeBolt(t)

	err := b.PutYear(2022, sample2022)
	require.NoError(t, err)

	f, err := os.Create(dir + "/backup.bolt")
	require.NoError(t, err)

	err = b.Backup(f)
	require.NoError(t, err)

	err = f.Close()
	require.NoError(t, err)
	err = b.Close()
	require.NoError(t, err)

	// Создать Bolt из бекапа и проверить, что все данные там.
	b, err = NewBolt(dir + "/backup.bolt")
	require.NoError(t, err)
	defer b.Close()

	y, ok := b.FindYear(2022)
	assert.True(t, ok)
	assert.Equal(t, sample2022, y)
}

func makeBolt(t *testing.T) (b *Bolt, dir string) {
	dir = t.TempDir()
	b, err := NewBolt(dir + "/db.bolt")
	require.NoError(t, err)
	return b, dir
}
