package collection

import (
	"github.com/stretchr/testify/require"
	"testing"
)

type mItem struct {
	key  string
	data int
}

func (m *mItem) GetKey() string {
	return m.key
}

func TestAdd(t *testing.T) {
	var ok bool
	assert := require.New(t)
	idxSlice := NewIndexingSlice[*mItem]()
	idxSlice.Add(&mItem{"test", 1})
	idxSlice.Add(&mItem{"test2", 2})
	idxSlice.Add(&mItem{"test3", 3})

	item, _ := idxSlice.GetByIndex(0)
	assert.Equal(1, item.data)
	item, _ = idxSlice.GetByKey("test2")
	assert.Equal(2, item.data)
	assert.Equal(3, idxSlice.Len())

	idxSlice.Delete("test")
	item, _ = idxSlice.GetByIndex(0)
	assert.Equal(2, idxSlice.Len())
	assert.Equal(3, item.data)

	idxSlice.Delete("test2")
	assert.Equal(1, idxSlice.Len())
	assert.Equal(1, idxSlice.LenKeys())

	item, ok = idxSlice.GetByKey("test")
	assert.Equal(false, ok)

	item, ok = idxSlice.GetByKey("test2")
	assert.Equal(false, ok)

	item, ok = idxSlice.GetByIndex(0)
	assert.Equal(3, item.data)
	item.data++
	item, ok = idxSlice.GetByIndex(0)
	assert.Equal(4, item.data)
}
